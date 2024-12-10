use std::error::Error;
use std::io::Error as IOError;

use store::Inode;

pub mod rocksdb;
pub mod store;

pub trait Store {
    fn new() -> Self;
    // TODO: we don't use create yet, this maybe used in training to save check points.
    fn create(&self, inode: Inode) -> Result<u64, Box<dyn Error>>;
    // read will try to search the file in the local path, if not found, query the siblings
    // for the file, if still not found, query the origin source for access and sync with its
    // siblings.
    fn read(&self, protocol_path: &str) -> Result<Vec<u8>, IOError>;
    fn delete(&self);
}
