use std::time::SystemTime;

use crate::store::Store;

pub struct Inode {
    // id could be different in different platforms, e.g. in huggingface, it's named
    // oid: 1ef325f1b111266a6b26e0196871bd78baa8c2f3, in object store, it's named ETAG.
    // id will be used to generate the uid as the identifier in stores.
    pub id: u64,
    pub path: String,
    pub size: u32,
    pub created_at: SystemTime,
    // pub updated_at: SystemTime,
    // pub last_visited_at: SystemTime,
}

impl Inode {}

pub struct BackendStore<T: Store> {
    store: T,
}

impl<T: Store> BackendStore<T> {
    pub fn new(store: T) -> Self {
        Self { store }
    }
}
