use std::error::Error;
use std::fs;
use std::io::Error as IOError;
use std::io::ErrorKind;
use std::path::Path;

use rocksdb::{Options, DB};

use crate::{store::Inode, store::Store, util::util::gen_inode};

const DB_PATH: &str = "/var/lib/rocksdb/manta";

pub struct Rocksdb {
    store: DB,
}

impl Store for Rocksdb {
    fn new() -> Self {
        let mut opts = Options::default();
        opts.create_if_missing(true);
        let db = DB::open(&opts, DB_PATH).expect("Failed to open the RocksDB");
        Self { store: db }
    }

    fn create(&self, inode: Inode) -> Result<u64, Box<dyn Error>> {
        // TODO: this is uncompleted.
        let id = gen_inode(&inode.id.to_string());
        self.store.put("inode", id.to_string())?;
        Ok(id)
    }

    // protocol_path looks like: s3:/a/b/c:version1, we should parse the address.
    // The real address should be $DB_PATH/s3/a/b/
    fn read(&self, protocol_path: &str) -> Result<Vec<u8>, IOError> {
        let patch_path = Path::new(DB_PATH).join(protocol_path);

        fs::read(patch_path).or_else(|err| {
            if err.kind() == ErrorKind::NotFound {
                // TODO: sync from other peers.
                Err(IOError::new(ErrorKind::NotFound, "File not exists"))
            } else {
                Err(err)
            }
        })
    }

    fn delete(&self) {}
}
