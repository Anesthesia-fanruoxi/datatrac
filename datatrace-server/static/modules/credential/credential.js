// 凭据管理核心逻辑
const CredentialCore = {
    // 加载凭据列表
    async loadList() {
        try {
            const response = await HttpUtils.get('/api/v1/credentials');
            const credentials = response.data || [];
            
            const tbody = document.getElementById('credentialList');
            if (credentials.length === 0) {
                tbody.innerHTML = `
                    <tr>
                        <td colspan="5" class="text-center text-muted">
                            暂无凭据，点击右上角"新建凭据"按钮创建
                        </td>
                    </tr>
                `;
                return;
            }
            
            tbody.innerHTML = credentials.map(cred => `
                <tr>
                    <td><strong>${cred.name}</strong></td>
                    <td>${cred.username}</td>
                    <td>${cred.description || '-'}</td>
                    <td>${FormatUtils.formatDate(cred.created_at)}</td>
                    <td>
                        <div class="credential-actions">
                            <button class="btn btn-sm btn-outline-primary" onclick="CredentialCore.showEdit('${cred.id}')">
                                <i class="bi bi-pencil"></i> 编辑
                            </button>
                            <button class="btn btn-sm btn-outline-danger" onclick="CredentialCore.confirmDelete('${cred.id}', '${cred.name}')">
                                <i class="bi bi-trash"></i> 删除
                            </button>
                        </div>
                    </td>
                </tr>
            `).join('');
        } catch (error) {
            console.error('加载凭据列表失败:', error);
            Toast.error('加载凭据列表失败');
        }
    },

    // 显示新建对话框
    showAdd() {
        document.getElementById('modalTitle').textContent = '新建凭据';
        document.getElementById('credentialId').value = '';
        document.getElementById('credentialForm').reset();
        document.getElementById('credentialModal').style.display = 'block';
    },

    // 显示编辑对话框
    async showEdit(id) {
        try {
            const response = await HttpUtils.get(`/api/v1/credentials/${id}`);
            const cred = response.data;
            
            document.getElementById('modalTitle').textContent = '编辑凭据';
            document.getElementById('credentialId').value = cred.id;
            document.getElementById('credName').value = cred.name;
            document.getElementById('credUsername').value = cred.username;
            document.getElementById('credPassword').value = ''; // 密码不回显
            document.getElementById('credDescription').value = cred.description || '';
            document.getElementById('credentialModal').style.display = 'block';
        } catch (error) {
            console.error('加载凭据详情失败:', error);
            Toast.error('加载凭据详情失败');
        }
    },

    // 关闭对话框
    closeModal() {
        document.getElementById('credentialModal').style.display = 'none';
        document.getElementById('credentialForm').reset();
    },

    // 保存凭据
    async save(event) {
        event.preventDefault();
        
        const id = document.getElementById('credentialId').value;
        const data = {
            name: document.getElementById('credName').value.trim(),
            username: document.getElementById('credUsername').value.trim(),
            password: document.getElementById('credPassword').value,
            description: document.getElementById('credDescription').value.trim()
        };

        // 编辑时如果密码为空，不传递密码字段
        if (id && !data.password) {
            delete data.password;
        }

        try {
            if (id) {
                await HttpUtils.put(`/api/v1/credentials/${id}`, data);
                Toast.success('更新成功');
            } else {
                await HttpUtils.post('/api/v1/credentials', data);
                Toast.success('创建成功');
            }
            
            this.closeModal();
            this.loadList();
        } catch (error) {
            console.error('保存凭据失败:', error);
            Toast.error(error.response?.data?.message || '保存失败');
        }
    },

    // 确认删除
    confirmDelete(id, name) {
        if (confirm(`确定要删除凭据"${name}"吗？\n\n注意：如果有数据源正在使用此凭据，将无法删除。`)) {
            this.delete(id);
        }
    },

    // 删除凭据
    async delete(id) {
        try {
            await HttpUtils.delete(`/api/v1/credentials/${id}`);
            Toast.success('删除成功');
            this.loadList();
        } catch (error) {
            console.error('删除凭据失败:', error);
            Toast.error(error.response?.data?.message || '删除失败');
        }
    }
};
