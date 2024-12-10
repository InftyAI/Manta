use std::{
    collections::hash_map::DefaultHasher,
    hash::{Hash, Hasher},
};

pub fn gen_inode(id: &str) -> u64 {
    let mut hasher = DefaultHasher::new();
    id.hash(&mut hasher);
    hasher.finish()
}

#[cfg(test)]
mod tests {
    use super::*;

    macro_rules! gen_inode_cases {
        ($($name:ident: $value:expr => $expected:expr,)*) => {
            $(
                #[test]
                fn $name() {
                    assert_eq!(gen_inode($value.0) == gen_inode($value.1), $expected)
                }
            )*
        };
    }

    gen_inode_cases! {
        test_same_path: ("1ef325f1b111266a6b26e0196871bd78baa8c2f3", "1ef325f1b111266a6b26e0196871bd78baa8c2f3") => true,
        test_different_path: ("1ef325f1b111266a6b26e0196871bd78baa8c2f3", "BC183D37238B0A274A915D5E035C9C0B") => false,
    }
}
