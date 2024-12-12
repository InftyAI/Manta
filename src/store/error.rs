use std::{error, fmt};

#[derive(Debug, Clone, PartialEq, Eq)]
pub struct Error {
    message: String,
}

impl Error {
    pub fn new(message: String) -> Error {
        Error { message }
    }
}

impl error::Error for Error {
    fn description(&self) -> &str {
        &self.message
    }
}

impl fmt::Display for Error {
    fn fmt(&self, formatter: &mut fmt::Formatter) -> fmt::Result {
        self.message.fmt(formatter)
    }
}

#[derive(Debug, Clone, PartialEq, Eq)]
pub enum ErrorKind {
    NotFound,
    Corruption,
    NotSupported,
    InvalidArgument,
    IOError,
    MergeInProgress,
    Incomplete,
    ShutdownInProgress,
    TimedOut,
    Aborted,
    Busy,
    Expired,
    TryAgain,
    CompactionTooLarge,
    ColumnFamilyDropped,
    Unknown,
}
