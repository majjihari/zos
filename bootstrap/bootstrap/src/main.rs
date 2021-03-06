#[macro_use]
extern crate failure;
#[macro_use]
extern crate log;

mod bootstrap;
mod config;
mod hub;
mod kparams;
mod workdir;
mod zfs;
mod zinit;

use config::Config;
use failure::Error;

type Result<T> = std::result::Result<T, Error>;

fn app() -> Result<()> {
    let config = Config::current()?;

    let level = if config.debug {
        log::Level::Debug
    } else {
        log::Level::Info
    };

    simple_logger::init_with_level(level).unwrap();

    // configure available stage
    let stages: Vec<fn(cfg: &Config) -> Result<()>> = vec![
        // self update
        |cfg| -> Result<()> {
            if cfg.debug {
                // if debug is set, do not upgrade self.
                return Ok(());
            }
            bootstrap::update(cfg)
        },
        // install all system binaries
        |cfg| -> Result<()> { bootstrap::install(cfg) },
        // install and start all services
        |cfg| -> Result<()> { bootstrap::bootstrap(cfg) },
    ];

    let index = config.stage as usize - 1;

    if index >= stages.len() {
        bail!(
            "unknown stage '{}' only {} stage(s) are supported",
            config.stage,
            stages.len()
        );
    }

    info!("running stage {}/{}", config.stage, stages.len());
    stages[index](&config)?;

    // Why we run stages in different "processes" (hence using exec)
    // the point is that will allow the stages to do changes to the
    // bootstrap binary. It means an old image with an old version of
    // bootstrap will still be able to run latest code. Because always
    // the first stage is to update self.
    let next = config.stage as usize + 1;
    if next <= stages.len() {
        debug!("spawning stage: {}", next);
        let bin: Vec<String> = std::env::args().take(1).collect();
        let mut cmd = exec::Command::new(&bin[0]);
        let cmd = cmd.arg("-s").arg(format!("{}", next));
        let cmd = if config.debug { cmd.arg("-d") } else { cmd };

        //this call will never return unless something is wrong.
        bail!("{}", cmd.exec());
    }

    Ok(())
}

fn main() {
    let code = match app() {
        Ok(_) => 0,
        Err(err) => {
            eprintln!("{}", err);
            1
        }
    };

    std::process::exit(code);
}
