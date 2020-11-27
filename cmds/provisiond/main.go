package main

import (
	"context"
	"flag"
	"os"
	"path/filepath"
	"time"

	"github.com/cenkalti/backoff/v3"
	"github.com/threefoldtech/zos/pkg"
	"github.com/threefoldtech/zos/pkg/app"
	"github.com/threefoldtech/zos/pkg/environment"
	"github.com/threefoldtech/zos/pkg/provision/explorer"
	"github.com/threefoldtech/zos/pkg/provision/primitives"
	"github.com/threefoldtech/zos/pkg/provision/primitives/cache"

	"github.com/threefoldtech/zos/pkg/stubs"
	"github.com/threefoldtech/zos/pkg/utils"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/threefoldtech/zbus"
	"github.com/threefoldtech/zos/pkg/provision"
	"github.com/threefoldtech/zos/pkg/version"
)

const (
	module = "provision"
)

func main() {
	app.Initialize()

	var (
		msgBrokerCon string
		storageDir   string
		debug        bool
		ver          bool
	)

	flag.StringVar(&storageDir, "root", "/var/cache/modules/provisiond", "root path of the module")
	flag.StringVar(&msgBrokerCon, "broker", "unix:///var/run/redis.sock", "connection string to the message broker")
	flag.BoolVar(&debug, "debug", false, "enable debug logging")
	flag.BoolVar(&ver, "v", false, "show version and exit")

	flag.Parse()
	if ver {
		version.ShowAndExit(false)
	}

	// Default level for this example is info, unless debug flag is present
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	if debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}

	// keep checking if limited-cache flag is set
	if app.CheckFlag(app.LimitedCache) {
		log.Error().Msg("failed cache reservation! Retrying every 30 seconds...")
		for app.CheckFlag(app.LimitedCache) {
			time.Sleep(time.Second * 30)
		}
	}

	flag.Parse()

	if err := os.MkdirAll(storageDir, 0770); err != nil {
		log.Fatal().Err(err).Msg("failed to create cache directory")
	}

	env, err := environment.Get()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse node environment")
	}

	if env.Orphan {
		// disable providiond on this node
		// we don't have a valid farmer id set
		log.Fatal().Msg("orphan node, we won't provision anything at all")
	}

	server, err := zbus.NewRedisServer(module, msgBrokerCon, 1)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to message broker")
	}
	zbusCl, err := zbus.NewRedisClient(msgBrokerCon)
	if err != nil {
		log.Fatal().Err(err).Msg("fail to connect to message broker server")
	}

	identity := stubs.NewIdentityManagerStub(zbusCl)
	nodeID := identity.NodeID()

	// block until networkd is ready to serve request from zbus
	// this is used to prevent uptime and online status to the explorer if the node is not in a fully ready
	// https://github.com/threefoldtech/zos/issues/632
	network := stubs.NewNetworkerStub(zbusCl)
	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = 0
	backoff.RetryNotify(func() error {
		return network.Ready()
	}, bo, func(err error, d time.Duration) {
		log.Error().Err(err).Msgf("networkd is not ready yet")
	})

	// to get reservation from tnodb
	explorerClient, err := app.ExplorerClient()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to instantiate BCDB client")
	}

	// keep track of resource units reserved and amount of workloads provisionned

	// to store reservation locally on the node
	store, err := cache.NewFSStore(filepath.Join(storageDir, "reservations"))
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create local reservation store")
	}

	const daemonBootFlag = "provisiond"
	if app.IsFirstBoot(daemonBootFlag) {
		if err := store.Purge(cache.NotPersisted); err != nil {
			log.Fatal().Err(err).Msg("failed to clean up cache")
		}
	}

	if err := app.MarkBooted(daemonBootFlag); err != nil {
		log.Fatal().Err(err).Msg("failed to mark service as booted")
	}

	// compatability fix
	if err := UpdateReservationsResults(store); err != nil {
		log.Fatal().Err(err).Msg("failed to upgrade cached reservations")
	}

	capacity, err := store.CurrentCounters()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to get current deployed capacity")
	}

	handlers := primitives.NewPrimitivesProvisioner(zbusCl)
	/* --- committer
	 *   --- cache
	 *	   --- statistics
	 *	     --- handlers
	 */
	provisioner := explorer.NewCommitterProvisioner(
		provision.NewCachedProvisioner(
			primitives.NewStatisticsProvisioner(
				handlers,
				capacity,
			),
			store,
		),
		explorerClient,
		primitives.ResultToSchemaType,
		identity,
		nodeID.Identity(),
	)

	puller := explorer.NewPoller(explorerClient, primitives.WorkloadToProvisionType, primitives.ProvisionOrder)
	engine := provision.New(provision.EngineOps{
		Source: provision.CombinedSource(
			provision.PollSource(puller, nodeID),
			provision.NewDecommissionSource(store),
		),
		Provisioner: provisioner,
		Janitor:     primitives.NewJanitor(zbusCl, puller),
	})

	server.Register(zbus.ObjectID{Name: module, Version: "0.0.1"}, pkg.Provision(engine))

	log.Info().
		Str("broker", msgBrokerCon).
		Msg("starting provision module")

	ctx := context.Background()
	ctx, _ = utils.WithSignal(ctx)
	utils.OnDone(ctx, func(_ error) {
		log.Info().Msg("shutting down")
	})

	// call the runtime upgrade before running engine
	handlers.RuntimeUpgrade(ctx)

	go func() {
		if err := server.Run(ctx); err != nil && err != context.Canceled {
			log.Fatal().Err(err).Msg("unexpected error")
		}
		log.Info().Msg("zbus server stopped")
	}()

	if err := engine.Run(ctx); err != nil {
		log.Error().Err(err).Msg("unexpected error")
	}
	log.Info().Msg("provision engine stopped")
}
