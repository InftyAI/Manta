use std::{io::Error, io::ErrorKind, time::SystemTime};

use async_trait::async_trait;
use sqlx::{sqlite::SqlitePool, Pool, Sqlite};

use crate::store::store::{Inode, Store, StoreType};

use super::store::InodeID;

static DB_PATH: &str = "var/lib/sqlite/manta.db";

pub struct SqliteStore {
    store: Pool<Sqlite>,
}

impl SqliteStore {
    pub async fn new() -> Self {
        let store = SqlitePool::connect(format!("sqlite://{}", DB_PATH).as_str())
            .await
            .unwrap();
        println!("Database connected successfully!");

        sqlx::query(
            "CREATE TABLE IF NOT EXISTS inodes (
                id INTEGER PRIMARY KEY AUTOINCREMENT,
                name TEXT NOT NULL,
                file_type TEXT NOT NULL CHECK(file_type in ('file', 'directory')),
                path TEXT NOT NULL,
                size INTEGER NOT NULL,
                parent_id INTEGER NOT NULL,
                storage_type TEXT NOT NULL CHECK(storage_type IN ('HF', 'MS', 'S3', 'OSS', 'GCS')),
                created_at DATETIME NOT NULL,
                updated_at DATETIME NOT NULL,
                last_visited_at DATETIME NOT NULL
                lock BOOLEAN NOT NULL DEFAULT 0,
            )",
        )
        .execute(&store)
        .await
        .unwrap();

        println!("Table inodes created successfully");
        Self { store }
    }
}

#[async_trait]
impl Store for SqliteStore {
    async fn get(&self, inode_id: InodeID) -> Result<Option<Inode>, Error> {
        let result = sqlx::query_as::<_, Inode>("SELECT * FROM inodes WHERE id = ?")
            .bind(inode_id.to_string())
            .fetch_optional(&self.store)
            .await
            .map_err(|e| Error::new(ErrorKind::Other, e.to_string()));

        result
    }

    async fn lookup(&self, parent_id: InodeID, name: &str) -> Result<Option<Inode>, Error> {
        let result =
            sqlx::query_as::<_, Inode>("SELECT * FROM inodes WHERE parent_id = ? AND name = ?")
                .bind(parent_id.to_string())
                .bind(name)
                .fetch_optional(&self.store)
                .await
                .map_err(|e| Error::new(ErrorKind::Other, e.to_string()));

        result
    }
}
