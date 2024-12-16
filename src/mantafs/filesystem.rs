use log::{debug, warn};
use std::{ffi::OsStr, time::Duration};

use fuser::{Filesystem, KernelConfig, ReplyAttr, ReplyData, ReplyEntry, Request, FUSE_ROOT_ID};
use libc::{c_int, EIO, ENOENT, ENOSYS};
use tokio::runtime::{self, Runtime};

use crate::store::store::{Backend, StoreType};

lazy_static::lazy_static! {
    static ref RUNTIME: Runtime = runtime::Builder::new_current_thread().enable_all().build().unwrap();
}

struct mantafs {
    backend: Backend,
}

impl mantafs {
    async fn new() -> Self {
        let backend = Backend::new(StoreType::Sqlite).await;
        Self { backend }
    }
}

impl Filesystem for mantafs {
    fn init(&mut self, _req: &Request, _config: &mut KernelConfig) -> Result<(), c_int> {
        Ok(())
    }

    fn lookup(&mut self, _req: &Request, parent: u64, name: &OsStr, reply: ReplyEntry) {
        match name.to_str() {
            Some(value) => {
                let result =
                    runtime::Runtime::block_on(&RUNTIME, self.backend.lookup(parent, value));

                // TODO: check access
                match result {
                    Ok(Some(inode)) => {
                        reply.entry(&Duration::new(0, 0), &inode.convert_to_fileAttr(), 0);
                    }
                    Ok(None) => {
                        reply.error(ENOENT);
                    }
                    Err(_err) => {
                        reply.error(EIO);
                    }
                }
            }
            None => {
                reply.error(ENOENT);
            }
        }
    }

    fn getattr(&mut self, _req: &Request<'_>, ino: u64, fh: Option<u64>, reply: ReplyAttr) {
        // let attr = fuser::FileAttr { ino, size, blocks };
        warn!(
            "[Not Implemented] getattr(ino: {:#x?}, fh: {:#x?})",
            ino, fh
        );
        reply.error(ENOSYS);
    }

    fn mkdir(
        &mut self,
        _req: &Request<'_>,
        parent: u64,
        name: &OsStr,
        mode: u32,
        umask: u32,
        reply: ReplyEntry,
    ) {
        debug!(
            "[Not Implemented] mkdir(parent: {:#x?}, name: {:?}, mode: {}, umask: {:#x?})",
            parent, name, mode, umask
        );
        reply.error(ENOSYS);
    }

    fn read(
        &mut self,
        _req: &Request<'_>,
        ino: u64,
        fh: u64,
        offset: i64,
        size: u32,
        flags: i32,
        lock_owner: Option<u64>,
        reply: ReplyData,
    ) {
        warn!(
            "[Not Implemented] read(ino: {:#x?}, fh: {}, offset: {}, size: {}, \
            flags: {:#x?}, lock_owner: {:?})",
            ino, fh, offset, size, flags, lock_owner
        );
        reply.error(ENOSYS);
    }
}
