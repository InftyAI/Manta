use log::{debug, warn};
use std::ffi::OsStr;

use fuser::{Filesystem, KernelConfig, ReplyAttr, ReplyData, ReplyEntry, Request, FUSE_ROOT_ID};
use libc::{c_int, ENOENT, ENOSYS};

use crate::store::store::{BackendStore, StoreType};

#[derive(Debug)]
struct mantafs {
    root: String,
    cache_dir: String,
}

impl mantafs {
    fn new(root: String, cache_dir: String, storage: String) -> Self {
        Self { root, cache_dir }
    }
}

impl Filesystem for mantafs {
    fn init(&mut self, _req: &Request, _config: &mut KernelConfig) -> Result<(), c_int> {
        let backend = BackendStore::new(StoreType::RocksDB);
    }

    fn lookup(&mut self, _req: &Request, parent: u64, name: &OsStr, reply: ReplyEntry) {
        warn!(
            "[Not Implemented] lookup(parent: {:#x?}, name {:?})",
            parent, name
        );
        reply.error(ENOENT);
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
