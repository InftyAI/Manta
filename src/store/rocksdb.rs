use std::fs;
use std::io::{Error, ErrorKind};
use std::path::Path;

use bincode;

use rocksdb::{Options, DB};

// use crate::util::util::parse_path;
use crate::{
    store::store::Inode, store::store::InodeID, store::store::Store, util::util::gen_inode,
};

const DB_PATH: &str = "/var/lib/rocksdb/manta";

pub struct Rocksdb {
    store: DB,
}

impl Rocksdb {
    pub fn new() -> Self {
        let mut opts = Options::default();
        opts.create_if_missing(true);
        let db = DB::open(&opts, DB_PATH).expect("Failed to open the RocksDB");
        Self { store: db }
    }
}

impl Store for Rocksdb {
    fn create(&self, _inode: Inode) -> Result<u64, Error> {
        Ok(0)
    }

    fn get(&self, _inode_id: InodeID) -> Result<Option<Inode>, Error> {
        Err(Error::new(
            ErrorKind::Unsupported,
            ErrorKind::Unsupported.to_string(),
        ))
        // match self.store.get(inode_id.to_le_bytes().to_vec()) {
        //     Ok(Some(value)) => {
        //         bincode::decode_from_reader(reader, config)
        //     },
        //     Ok(None) => Err(Error::new(
        //         ErrorKind::NotFound,
        //         ErrorKind::NotFound.to_string(),
        //     )),
        //     Err(err) => Err(Error::new(ErrorKind::Other, err.into_string())),
        // }
    }

    // protocol_path looks like: s3://a/b/c.txt:v1 or hf://Qwen/Qwen2.5-72B-Instruct:main,
    // we should parse the address. The real address should be $DB_PATH/s3/a/b/c.txt or
    // $DB_PATH/hf/Qwen/Qwen2.5-72B-Instruct.
    // TODO: handle version controller in object stores.
    // fn read(&self, protocol_path: &str) -> Result<Vec<u8>, IOError> {
    //     match parse_path(protocol_path) {
    //         // TODO: version is not used yet.
    //         Ok((protocol, path, _version)) => {
    //             let fixed_path = if path.starts_with("/") {
    //                 path.to_string()
    //             } else {
    //                 format!("/{}", path)
    //             };

    //             let patch_path = Path::new(DB_PATH)
    //                 .join(format!("/{}", protocol))
    //                 .join(fixed_path);

    //             fs::read(patch_path).or_else(|err| {
    //                 if err.kind() == ErrorKind::NotFound {
    //                     // TODO: sync from other peers.
    //                     Err(IOError::new(ErrorKind::NotFound, "File not exists"))
    //                 } else {
    //                     Err(err)
    //                 }
    //             })
    //         }
    //         Err(err) => Err(IOError::new(ErrorKind::Other, err)),
    //     }
    // }

    fn delete(&self) {}
}
