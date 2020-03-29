package builder

import (
	"encoding/json"
	"time"

	"github.com/threefoldtech/zos/pkg/schema"
	"github.com/threefoldtech/zos/tools/explorer/models/generated/workloads"
)

const (
	defaultReservationDuration = 30 * time.Hour * 24 //30 days
)

// ReservationBuilder struct
type ReservationBuilder struct {
	user        int64
	signer      *Signer
	reservation workloads.Reservation
}

// NewReservationBuilder creates a new reservation builder
func NewReservationBuilder(user int64, signer *Signer) *ReservationBuilder {
	expire := time.Now().Add(defaultReservationDuration)
	return &ReservationBuilder{
		user:   user,
		signer: signer,
		reservation: workloads.Reservation{
			DataReservation: workloads.ReservationData{
				ExpirationReservation: schema.Date{Time: expire},
			},
		},
	}
}

// WithExpiration sets expiration
func (b *ReservationBuilder) WithExpiration(t time.Time) *ReservationBuilder {
	b.reservation.DataReservation.ExpirationReservation = schema.Date{Time: t}
	return b
}

// AddVolume adds a new workload to the reservations
func (b *ReservationBuilder) AddVolume(builder VolumeBuilder) {
	b.reservation.DataReservation.Volumes = append(b.reservation.DataReservation.Volumes, builder.Build())
}

// Build returns a reservation object
func (b *ReservationBuilder) Build() (workloads.Reservation, error) {
	b.reservation.CustomerTid = b.user
	data, err := json.Marshal(b.reservation.DataReservation)
	if err != nil {
		return b.reservation, err
	}
	b.reservation.Json = string(data)
	b.reservation.Epoch = schema.Date{Time: time.Now()}
	b.reservation.DataReservation.ExpirationProvisioning = schema.Date{Time: time.Now().Add(20 * time.Minute)}

	// if no delete signers were added, we must add the user himself
	if len(b.reservation.DataReservation.SigningRequestDelete.Signers) == 0 {
		b.reservation.DataReservation.SigningRequestDelete.Signers = []int64{b.user}
		b.reservation.DataReservation.SigningRequestDelete.QuorumMin = 1
	}

	_, signature, err := b.signer.SignHex(b.reservation.Json)
	b.reservation.CustomerSignature = signature

	return b.reservation, nil
}
