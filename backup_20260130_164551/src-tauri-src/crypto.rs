// 加密服务模块
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

/// 加密服务，负责密码等敏感信息的加密和解密
pub struct CryptoService {
    cipher: Aes256Gcm,
}

impl CryptoService {
    /// 创建新的加密服务实例
    /// 
    /// 首次运行时会生成随机密钥并保存到本地文件
    /// 后续运行会从文件加载已有密钥
    pub fn new() -> Result<Self> {
        let key = Self::load_or_generate_key()?;
        let cipher = Aes256Gcm::new(&key.into());
        
        Ok(Self { cipher })
    }

    /// 加密明文字符串
    /// 
    /// # 参数
    /// * `plaintext` - 要加密的明文
    /// 
    /// # 返回
    /// Base64 编码的密文，格式为 "nonce:ciphertext"
    pub fn encrypt(&self, plaintext: &str) -> Result<String> {
        // 生成随机 nonce (96 位 = 12 字节)
        let mut nonce_bytes = [0u8; 12];
        OsRng.fill_bytes(&mut nonce_bytes);
        let nonce = Nonce::from_slice(&nonce_bytes);

        // 加密
        let ciphertext = self
            .cipher
            .encrypt(nonce, plaintext.as_bytes())
            .map_err(|e| anyhow::anyhow!("加密失败: {}", e))?;

        // 编码为 Base64: nonce:ciphertext
        let nonce_b64 = STANDARD.encode(&nonce_bytes);
        let ciphertext_b64 = STANDARD.encode(&ciphertext);
        
        Ok(format!("{}:{}", nonce_b64, ciphertext_b64))
    }

    /// 解密密文字符串
    /// 
    /// # 参数
    /// * `ciphertext` - Base64 编码的密文，格式为 "nonce:ciphertext"
    /// 
    /// # 返回
    /// 解密后的明文
    pub fn decrypt(&self, ciphertext: &str) -> Result<String> {
        // 解析 nonce 和 ciphertext
        let parts: Vec<&str> = ciphertext.split(':').collect();
        if parts.len() != 2 {
            anyhow::bail!("密文格式错误");
        }

        let nonce_bytes = STANDARD.decode(parts[0])
            .context("nonce Base64 解码失败")?;
        let ciphertext_bytes = STANDARD.decode(parts[1])
            .context("密文 Base64 解码失败")?;

        if nonce_bytes.len() != 12 {
            anyhow::bail!("nonce 长度错误");
        }

        let nonce = Nonce::from_slice(&nonce_bytes);

        // 解密
        let plaintext_bytes = self
            .cipher
            .decrypt(nonce, ciphertext_bytes.as_ref())
            .map_err(|e| anyhow::anyhow!("解密失败: {}", e))?;

        String::from_utf8(plaintext_bytes)
            .context("解密后的数据不是有效的 UTF-8")
    }

    /// 加载或生成加密密钥
    /// 
    /// 密钥存储在用户目录下的 .datasync/encryption.key 文件中
    /// 如果文件不存在，则生成新的随机密钥并保存
    fn load_or_generate_key() -> Result<[u8; 32]> {
        let key_path = Self::get_key_path()?;

        if key_path.exists() {
            // 从文件加载密钥
            let key_data = fs::read(&key_path)
                .context("读取密钥文件失败")?;
            
            if key_data.len() != 32 {
                anyhow::bail!("密钥文件长度错误，期望 32 字节");
            }

            let mut key = [0u8; 32];
            key.copy_from_slice(&key_data);
            
            Ok(key)
        } else {
            // 生成新密钥
            let mut key = [0u8; 32];
            OsRng.fill_bytes(&mut key);

            // 确保目录存在
            if let Some(parent) = key_path.parent() {
                fs::create_dir_all(parent)
                    .context("创建密钥目录失败")?;
            }

            // 保存密钥到文件
            fs::write(&key_path, &key)
                .context("保存密钥文件失败")?;

            Ok(key)
        }
    }

    /// 获取密钥文件路径
    fn get_key_path() -> Result<PathBuf> {
        let home_dir = dirs::home_dir()
            .ok_or_else(|| anyhow::anyhow!("无法获取用户主目录"))?;
        
        Ok(home_dir.join(".datasync").join("encryption.key"))
    }
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_encrypt_decrypt_roundtrip() {
        let crypto = CryptoService::new().unwrap();
        let plaintext = "my_password_123";
        
        let encrypted = crypto.encrypt(plaintext).unwrap();
        let decrypted = crypto.decrypt(&encrypted).unwrap();
        
        assert_eq!(plaintext, decrypted);
    }

    #[test]
    fn test_encrypt_produces_different_ciphertext() {
        let crypto = CryptoService::new().unwrap();
        let plaintext = "same_password";
        
        let encrypted1 = crypto.encrypt(plaintext).unwrap();
        let encrypted2 = crypto.encrypt(plaintext).unwrap();
        
        // 由于使用随机 nonce，每次加密结果应该不同
        assert_ne!(encrypted1, encrypted2);
        
        // 但解密后应该相同
        assert_eq!(crypto.decrypt(&encrypted1).unwrap(), plaintext);
        assert_eq!(crypto.decrypt(&encrypted2).unwrap(), plaintext);
    }

    #[test]
    fn test_decrypt_invalid_format() {
        let crypto = CryptoService::new().unwrap();
        
        // 缺少冒号分隔符
        let result = crypto.decrypt("invalid_format");
        assert!(result.is_err());
        
        // 只有一个部分
        let result = crypto.decrypt("only_one_part:");
        assert!(result.is_err());
    }

    #[test]
    fn test_decrypt_invalid_base64() {
        let crypto = CryptoService::new().unwrap();
        
        // 无效的 Base64
        let result = crypto.decrypt("!!!:@@@");
        assert!(result.is_err());
    }
}
