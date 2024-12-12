use std::time::SystemTime;

use crate::store::rocksdb::Rocksdb;
use std::io::Error;

pub type InodeID = u64;

pub trait Store {
    fn create(&self, inode: Inode) -> Result<u64, Error>;
    fn get(&self, inode_id: InodeID) -> Result<Option<Inode>, Error>;
    // read will try to search the file in the local path, if not found, query the siblings
    // for the file, if still not found, query the origin source for access and sync with its
    // siblings.
    // fn read(&self, protocol_path: &str) -> Result<Vec<u8>, IOError>;
    fn delete(&self);
}

pub struct Inode {
    id: InodeID,
    // ident_hash could be different in different platforms, e.g. in huggingface, it's named
    // oid: 1ef325f1b111266a6b26e0196871bd78baa8c2f3, in object store, it's named ETAG.
    // id will be used to generate the uid as the identifier in stores.
    ident_hash: String,
    path: String,
    size: u32,
    created_at: SystemTime,
    updated_at: SystemTime,
    last_visited_at: SystemTime,
}

pub enum StoreType {
    RocksDB,
}

// The facade for Store trait.
pub struct BackendStore {
    store: Box<dyn Store>,
}

impl BackendStore {
    pub fn new(store_type: StoreType) -> Self {
        let store: Box<dyn Store> = match store_type {
            StoreType::RocksDB => Box::new(Rocksdb::new()),
        };
        Self { store }
    }
}
