use std::{
    collections::hash_map::DefaultHasher,
    hash::{Hash, Hasher},
    path,
};

pub fn gen_inode(id: &str) -> u64 {
    let mut hasher = DefaultHasher::new();
    id.hash(&mut hasher);
    hasher.finish()
}

// protocol_path looks like: s3://a/b/c:v1, hf://Qwen/Qwen2.5-72B-Instruct:main.
pub fn parse_path(protocol_path: &str) -> Result<(&str, &str, &str), &'static str> {
    let paths: Vec<&str> = protocol_path.split("://").collect();
    if paths.len() != 2 {
        return Err("protocol not right");
    }
    let splits: Vec<&str> = paths[1].split(":").collect();
    let version = if splits.len() == 2 { splits[1] } else { "" };
    Ok((paths[0], splits[0], version))
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

    macro_rules! parse_cases {
        ($($name:ident: $value:expr => $expected:expr,)*) => {
            $(
                #[test]
                fn $name() {
                    assert_eq!(parse_path($value), $expected)
                }
            ) *
        };
    }

    parse_cases! {
        test_s3_path: "s3://a/b/c:v1" => Ok(("s3", "a/b/c", "v1")),
        test_oss_without_version: "oss://a/b/c" => Ok(("oss", "a/b/c", "")),
        test_hf: "hf://Qwen/Qwen2.5-72B-Instruct:main" => Ok(("hf", "Qwen/Qwen2.5-72B-Instruct", "main")),
        test_err_with_err_protocol: "hf:Qwen/Qwen2.5-72B-Instruct:main" => Err("protocol not right"),
    }
}
