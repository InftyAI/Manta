use std::io::Error;

use async_trait::async_trait;
use chrono::NaiveDateTime;
use fuser::{self, FileAttr};
use sqlx::FromRow;

use crate::util::util;

use super::sqlite::SqliteStore;

pub type InodeID = u64;

#[async_trait]
pub trait Store {
    async fn get(&self, inode_id: InodeID) -> Result<Option<Inode>, Error>;
    async fn lookup(&self, parent_id: InodeID, name: &str) -> Result<Option<Inode>, Error>;
    // fn create(&self, inode: Inode) -> Result<u64, Error>;
    // // read will try to search the file in the local path, if not found, query the siblings
    // // for the file, if still not found, query the origin source for access and sync with its
    // // siblings.
    // // fn read(&self, protocol_path: &str) -> Result<Vec<u8>, IOError>;
    // fn delete(&self);
}

#[derive(Debug, FromRow)]
pub struct Inode {
    pub id: InodeID,
    pub name: String,
    pub path: String,
    pub size: u64,
    pub parent_id: u64,
    pub store_type: StoreType,
    pub kind_type: KindType,
    pub created_at: NaiveDateTime,
    pub updated_at: NaiveDateTime,
    pub last_visited_at: NaiveDateTime,
    pub lock: bool,
}

impl Inode {
    pub fn convert_to_fileAttr(&self) -> fuser::FileAttr {
        fuser::FileAttr {
            ino: self.id,
            size: self.size,
            blocks: 0,
            atime: util::naive_to_system_time(self.last_visited_at),
            mtime: util::naive_to_system_time(self.updated_at),
            ctime: util::naive_to_system_time(self.updated_at),
            crtime: util::naive_to_system_time(self.created_at),
            kind: self.kind_type.clone().into(),
            perm: 0777,
            nlink: 0,
            uid: 1000,
            gid: 1000,
            rdev: 0,
            blksize: 4096,
            flags: 0,
        }
    }
}

#[derive(Debug, sqlx::Type)]
pub enum StoreType {
    Sqlite,
}

#[derive(Debug, Clone, sqlx::Type)]
pub enum KindType {
    Directory,
    File,
    Symlink,
}

impl From<KindType> for fuser::FileType {
    fn from(kind: KindType) -> Self {
        match kind {
            KindType::Directory => fuser::FileType::Directory,
            KindType::File => fuser::FileType::RegularFile,
            KindType::Symlink => fuser::FileType::Symlink,
        }
    }
}

// The facade for Store trait.
pub struct Backend {
    store: Box<dyn Store>,
}

impl Backend {
    pub async fn new(store_type: StoreType) -> Self {
        let store: Box<dyn Store> = match store_type {
            StoreType::Sqlite => Box::new(SqliteStore::new().await),
        };
        Self { store }
    }

    pub async fn get(&self, inode_id: InodeID) -> Result<Option<Inode>, Error> {
        self.store.get(inode_id).await
    }

    pub async fn lookup(&self, parent_id: InodeID, name: &str) -> Result<Option<Inode>, Error> {
        self.store.lookup(parent_id, name).await
    }
}
