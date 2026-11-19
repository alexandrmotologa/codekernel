pub trait ChecksumProvider {
    fn compute_hash(&self, input: &[u8]) -> u64;
}

pub struct FnvHasher {
    offset_basis: u64,
}

impl FnvHasher {
    pub fn new() -> Self {
        FnvHasher {
            offset_basis: 0xcbf29ce484222325,
        }
    }
}

impl ChecksumProvider for FnvHasher {
    fn compute_hash(&self, input: &[u8]) -> u64 {
        let mut hash = self.offset_basis;
        for byte in input {
            hash ^= *byte as u64;
            hash = hash.wrapping_mul(0x100000001b3);
        }
        hash
    }
}
