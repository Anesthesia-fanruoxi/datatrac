// 凭据管理模块入口
const CredentialModule = {
    init() {
        console.log('凭据管理模块初始化...');
        
        // 加载凭据列表
        CredentialCore.loadList();
        
        // 绑定表单提交事件
        const form = document.getElementById('credentialForm');
        if (form) {
            form.addEventListener('submit', (e) => CredentialCore.save(e));
        }
        
        // 点击模态框外部关闭
        const modal = document.getElementById('credentialModal');
        if (modal) {
            modal.addEventListener('click', (e) => {
                if (e.target === modal) {
                    CredentialCore.closeModal();
                }
            });
        }
        
        console.log('✅ 凭据管理模块初始化完成');
    }
};
