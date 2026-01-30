// 加密服务
// 使用 AES-256-GCM 加密算法保护敏感信息

use aes_gcm::{
    aead::{Aead, KeyInit, OsRng},
    Aes256Gcm, Nonce,
};
use anyhow::{Context, Result};
use base64::{engine::general_purpose::STANDARD, Engine as _};
use rand::RngCore;
use std::fs;
use std::path::PathBuf;

/// 加密服务
pub struct CryptoService {
    cipher: Aes256Gcm,
}

impl CryptoService {
    /// 创建新的加密服务实例
    pub fn new() -> Result<Self> {
        let key = Self::load_or_generate_key()?;
        let cipher = Aes256Gcm::new(&key.into());
        Ok(Self { cipher })
    }

    /// 加密明文字符串
    pub fn encrypt(&self, plaintext: &str) -> Result<String> {
        let mut nonce_bytes = [0u8; 12];
        OsRng.fill_bytes(&mut nonce_bytes);
        let nonce = Nonce::from_slice(&nonce_bytes);

        let ciphertext = self
            .cipher
            .encrypt(nonce, plaintext.as_bytes())
            .map_err(|e| anyhow::anyhow!("加密失败: {}", e))?;

        let nonce_b64 = STANDARD.encode(&nonce_bytes);
        let ciphertext_b64 = STANDARD.encode(&ciphertext);

        Ok(format!("{}:{}", nonce_b64, ciphertext_b64))
    }

    /// 解密密文字符串
    pub fn decrypt(&self, ciphertext: &str) -> Result<String> {
        let parts: Vec<&str> = ciphertext.split(':').collect();
        if parts.len() != 2 {
            anyhow::bail!("密文格式错误");
        }

        let nonce_bytes = STANDARD
            .decode(parts[0])
            .context("nonce Base64 解码失败")?;
        let ciphertext_bytes = STANDARD
            .decode(parts[1])
            .context("密文 Base64 解码失败")?;

        if nonce_bytes.len() != 12 {
            anyhow::bail!("nonce 长度错误");
        }

        let nonce = Nonce::from_slice(&nonce_bytes);

        let plaintext_bytes = self
            .cipher
            .decrypt(nonce, ciphertext_bytes.as_ref())
            .map_err(|e| anyhow::anyhow!("解密失败: {}", e))?;

        String::from_utf8(plaintext_bytes).context("解密后的数据不是有效的 UTF-8")
    }

    /// 加载或生成加密密钥
    fn load_or_generate_key() -> Result<[u8; 32]> {
        let key_path = Self::get_key_path()?;

        if key_path.exists() {
            let key_data = fs::read(&key_path).context("读取密钥文件失败")?;

            if key_data.len() != 32 {
                anyhow::bail!("密钥文件长度错误，期望 32 字节");
            }

            let mut key = [0u8; 32];
            key.copy_from_slice(&key_data);

            Ok(key)
        } else {
            let mut key = [0u8; 32];
            OsRng.fill_bytes(&mut key);

            if let Some(parent) = key_path.parent() {
                fs::create_dir_all(parent).context("创建密钥目录失败")?;
            }

            fs::write(&key_path, &key).context("保存密钥文件失败")?;

            Ok(key)
        }
    }

    /// 获取密钥文件路径
    fn get_key_path() -> Result<PathBuf> {
        let home_dir = dirs::home_dir().ok_or_else(|| anyhow::anyhow!("无法获取用户主目录"))?;
        Ok(home_dir.join(".datatrac").join("encryption.key"))
    }
}
