// 数据源核心逻辑
(function() {
    'use strict';
    
    window.DataSourceCore = {
        currentId: null,
        eventSource: null,
        
        // 加载数据源列表
        loadList: async function() {
            try {
                const result = await HttpUtils.get('/api/v1/datasources');
                const tbody = document.getElementById('datasourceList');
                
                if (!result.data || result.data.length === 0) {
                    tbody.innerHTML = '<tr><td colspan="5" class="text-center text-muted">暂无数据源</td></tr>';
                    return;
                }
                
                tbody.innerHTML = result.data.map(ds => `
                    <tr>
                        <td>
                            <i class="bi bi-${ds.type === 'mysql' ? 'database-fill' : 'search'} me-2" style="color: #667eea;"></i>
                            <strong>${ds.name}</strong>
                        </td>
                        <td><span class="badge bg-primary">${ds.type === 'mysql' ? 'MySQL' : 'Elasticsearch'}</span></td>
                        <td><code>${ds.host}:${ds.port}</code></td>
                        <td id="status-${ds.id}">
                            <span class="text-muted">
                                <i class="bi bi-hourglass-split"></i> 等待测试...
                            </span>
                        </td>
                        <td>
                            <button class="btn btn-sm btn-outline-primary" onclick="DataSourceCore.edit('${ds.id}')">
                                <i class="bi bi-pencil"></i> 编辑
                            </button>
                            <button class="btn btn-sm btn-outline-danger" onclick="DataSourceCore.delete('${ds.id}', '${ds.name}')">
                                <i class="bi bi-trash"></i> 删除
                            </button>
                        </td>
                    </tr>
                `).join('');
                
                // 启动 SSE 测试
                if (window.DataSourceSSE) {
                    window.DataSourceSSE.start();
                }
            } catch (error) {
                console.error('加载数据源失败:', error);
                Toast.error('加载数据源失败: ' + error.message);
            }
        },

        // 显示新建表单
        showAdd: function() {
            document.getElementById('datasourceModal').style.display = 'block';
            document.getElementById('datasourceForm').reset();
            document.getElementById('datasourceId').value = '';
            document.getElementById('modalTitle').textContent = '新建数据源';
            
            // 重置认证方式为手动输入
            const manualRadio = document.getElementById('authTypeManual');
            if (manualRadio) manualRadio.checked = true;
            this.toggleAuthType();
            
            // 设置密码为必填
            const passwordField = document.getElementById('dsPassword');
            if (passwordField) {
                passwordField.required = true;
                passwordField.placeholder = '请输入密码';
            }
            
            // 添加ESC键关闭功能
            this.modalEscHandler = (e) => {
                if (e.key === 'Escape') {
                    this.closeModal();
                }
            };
            document.addEventListener('keydown', this.modalEscHandler);
        },
        
        // 编辑数据源
        edit: async function(id) {
            try {
                const result = await HttpUtils.get(`/api/v1/datasources/${id}`);
                
                if (result.code === 200) {
                    const ds = result.data;
                    
                    // 先显示模态框
                    const modal = document.getElementById('datasourceModal');
                    if (modal) modal.style.display = 'block';
                    
                    // 安全设置值
                    const setValueSafe = (id, value) => {
                        const el = document.getElementById(id);
                        if (el) el.value = value || '';
                    };
                    
                    setValueSafe('datasourceId', ds.id);
                    setValueSafe('dsName', ds.name);
                    setValueSafe('dsType', ds.type);
                    setValueSafe('dsHost', ds.host);
                    setValueSafe('dsPort', ds.port);
                    setValueSafe('dsDatabase', ds.database_name);
                    
                    // 判断是使用凭据还是手动输入
                    if (ds.credential_id) {
                        const credRadio = document.getElementById('authTypeCredential');
                        if (credRadio) credRadio.checked = true;
                        setValueSafe('dsCredential', ds.credential_id);
                        
                        // 加载凭据信息显示名称
                        try {
                            const credResult = await HttpUtils.get(`/api/v1/credentials/${ds.credential_id}`);
                            if (credResult.code === 200 && credResult.data) {
                                setValueSafe('dsCredentialName', `${credResult.data.name} (${credResult.data.username})`);
                            }
                        } catch (e) {
                            setValueSafe('dsCredentialName', '凭据ID: ' + ds.credential_id);
                        }
                    } else {
                        const manualRadio = document.getElementById('authTypeManual');
                        if (manualRadio) manualRadio.checked = true;
                        setValueSafe('dsUsername', ds.username);
                        setValueSafe('dsPassword', '');
                        
                        const passwordField = document.getElementById('dsPassword');
                        if (passwordField) {
                            passwordField.placeholder = '留空表示不修改';
                            passwordField.required = false;
                        }
                    }
                    
                    this.toggleAuthType();
                    
                    const typeSelect = document.getElementById('dsType');
                    if (typeSelect) typeSelect.dispatchEvent(new Event('change'));
                    
                    const modalTitle = document.getElementById('modalTitle');
                    if (modalTitle) modalTitle.textContent = '编辑数据源';
                    
                    // 添加ESC键关闭功能
                    this.modalEscHandler = (e) => {
                        if (e.key === 'Escape') {
                            this.closeModal();
                        }
                    };
                    document.addEventListener('keydown', this.modalEscHandler);
                } else {
                    Toast.error('加载失败: ' + result.message);
                }
            } catch (error) {
                console.error('编辑数据源错误:', error);
                Toast.error('加载失败: ' + error.message);
            }
        },

        // 删除数据源
        delete: function(id, name) {
            Modal.confirm(
                `确定要删除数据源"${name}"吗？<br><span class="text-danger">此操作不可恢复！</span>`,
                async () => {
                    try {
                        const result = await HttpUtils.delete(`/api/v1/datasources/${id}`);
                        
                        if (result.code === 200) {
                            Toast.success('删除成功！');
                            this.loadList();
                        } else {
                            Toast.error('删除失败: ' + result.message);
                        }
                    } catch (error) {
                        Toast.error('删除失败: ' + error.message);
                    }
                }
            );
        },
        
        // 关闭模态框
        closeModal: function() {
            document.getElementById('datasourceModal').style.display = 'none';
            // 移除ESC键监听
            if (this.modalEscHandler) {
                document.removeEventListener('keydown', this.modalEscHandler);
                this.modalEscHandler = null;
            }
        },
        
        // 测试连接
        testConnection: async function() {
            const authTypeRadio = document.querySelector('input[name="authType"]:checked');
            const authType = authTypeRadio ? authTypeRadio.value : 'manual';

            const data = {
                type: document.getElementById('dsType').value,
                host: document.getElementById('dsHost').value,
                port: parseInt(document.getElementById('dsPort').value, 10) || 0,
                database_name: (document.getElementById('dsDatabase') && document.getElementById('dsDatabase').value) || ''
            };

            if (!data.type || !data.host || !data.port) {
                Toast.warning('请填写类型、主机和端口');
                return;
            }

            if (authType === 'credential') {
                const credentialId = document.getElementById('dsCredential') && document.getElementById('dsCredential').value;
                if (!credentialId) {
                    Toast.warning('请选择凭据');
                    return;
                }
                data.credential_id = credentialId;
            } else {
                data.username = (document.getElementById('dsUsername') && document.getElementById('dsUsername').value) || '';
                data.password = (document.getElementById('dsPassword') && document.getElementById('dsPassword').value) || '';
                if (!data.username || !data.password) {
                    Toast.warning('请填写用户名和密码');
                    return;
                }
            }

            try {
                Toast.info('正在测试连接...');
                const result = await HttpUtils.post('/api/v1/datasources/test', data);
                if (result.code === 200) {
                    Toast.success('连接成功！版本: ' + (result.data.version || '未知'));
                } else {
                    Toast.error('连接失败: ' + result.message);
                }
            } catch (error) {
                Toast.error('连接失败: ' + (error.message || ''));
            }
        },

        // 提交表单
        submitForm: async function(e) {
            e.preventDefault();
            
            const id = document.getElementById('datasourceId').value;
            const authTypeRadio = document.querySelector('input[name="authType"]:checked');
            const authType = authTypeRadio ? authTypeRadio.value : 'manual';
            
            const data = {
                name: document.getElementById('dsName').value,
                type: document.getElementById('dsType').value,
                host: document.getElementById('dsHost').value,
                port: parseInt(document.getElementById('dsPort').value),
                database_name: document.getElementById('dsDatabase').value
            };

            // 根据认证方式设置不同的字段
            if (authType === 'credential') {
                const credentialId = document.getElementById('dsCredential').value;
                if (!credentialId) {
                    Toast.warning('请选择凭据');
                    return;
                }
                data.credential_id = credentialId;
            } else {
                data.username = document.getElementById('dsUsername').value;
                data.password = document.getElementById('dsPassword').value;
                
                if (!data.username) {
                    Toast.warning('请输入用户名');
                    return;
                }
                if (!id && !data.password) {
                    Toast.warning('请输入密码');
                    return;
                }
            }

            try {
                const url = id ? `/api/v1/datasources/${id}` : '/api/v1/datasources';
                const result = id ? await HttpUtils.put(url, data) : await HttpUtils.post(url, data);
                
                if (result.code === 200) {
                    Toast.success(id ? '更新成功！' : '创建成功！');
                    this.closeModal();
                    this.loadList();
                } else {
                    Toast.error('操作失败: ' + result.message);
                }
            } catch (error) {
                Toast.error('操作失败: ' + error.message);
            }
        },
        
        // 切换认证方式
        toggleAuthType: function() {
            const authTypeRadio = document.querySelector('input[name="authType"]:checked');
            if (!authTypeRadio) {
                console.error('未找到选中的认证方式');
                return;
            }
            
            const authType = authTypeRadio.value;
            const credentialGroup = document.getElementById('credentialGroup');
            const manualAuthGroup = document.getElementById('manualAuthGroup');
            
            if (!credentialGroup || !manualAuthGroup) {
                console.error('credentialGroup或manualAuthGroup元素不存在');
                return;
            }
            
            if (authType === 'credential') {
                credentialGroup.style.display = 'block';
                manualAuthGroup.style.display = 'none';
                // 清空手动输入的字段
                const usernameEl = document.getElementById('dsUsername');
                const passwordEl = document.getElementById('dsPassword');
                if (usernameEl) usernameEl.value = '';
                if (passwordEl) passwordEl.value = '';
            } else {
                credentialGroup.style.display = 'none';
                manualAuthGroup.style.display = 'block';
                // 清空凭据选择
                const credentialEl = document.getElementById('dsCredential');
                const credentialNameEl = document.getElementById('dsCredentialName');
                if (credentialEl) credentialEl.value = '';
                if (credentialNameEl) credentialNameEl.value = '';
            }
        },

        // ========== 凭据选择器相关方法 ==========
        
        // 显示凭据选择器
        showCredentialSelector: function() {
            document.getElementById('credentialSelectModal').style.display = 'flex';
            this.loadCredentialSelectList();
        },
        
        // 关闭凭据选择器
        closeCredentialSelectModal: function() {
            document.getElementById('credentialSelectModal').style.display = 'none';
        },
        
        // 加载凭据列表
        loadCredentialSelectList: async function() {
            const tbody = document.getElementById('credentialSelectList');
            if (!tbody) return;
            
            try {
                const result = await HttpUtils.get('/api/v1/credentials');
                const list = (result.code === 200 && result.data) ? result.data : [];
                
                if (list.length === 0) {
                    tbody.innerHTML = '<tr><td colspan="4" class="text-center text-muted py-4">暂无凭据，点击上方「添加」创建</td></tr>';
                    return;
                }
                
                const self = this;
                tbody.innerHTML = list.map(cred => {
                    const name = this.escapeHtml(cred.name || '');
                    const user = this.escapeHtml(cred.username || '');
                    const desc = this.escapeHtml(cred.description || '-');
                    const id = this.escapeHtml(cred.id || '');
                    return `<tr data-cred-id="${id}" data-cred-name="${name}" data-cred-user="${user}">
                        <td><strong class="text-dark">${name}</strong></td>
                        <td><span class="text-secondary">${user}</span></td>
                        <td class="text-muted small">${desc}</td>
                        <td class="text-end">
                            <div class="credential-actions">
                                <button type="button" class="btn btn-sm btn-outline-primary credential-btn-select" title="选择"><i class="bi bi-check-lg"></i></button>
                                <button type="button" class="btn btn-sm btn-outline-secondary credential-btn-edit" title="编辑"><i class="bi bi-pencil"></i></button>
                                <button type="button" class="btn btn-sm btn-outline-danger credential-btn-delete" title="删除"><i class="bi bi-trash"></i></button>
                            </div>
                        </td>
                    </tr>`;
                }).join('');
                tbody.querySelectorAll('.credential-btn-select').forEach(btn => {
                    btn.addEventListener('click', function() {
                        const tr = this.closest('tr');
                        if (tr) self.selectCredential(tr.dataset.credId, tr.dataset.credName, tr.dataset.credUser);
                    });
                });
                tbody.querySelectorAll('.credential-btn-edit').forEach(btn => {
                    btn.addEventListener('click', function() {
                        const tr = this.closest('tr');
                        if (tr) self.showEditCredentialForm(tr.dataset.credId);
                    });
                });
                tbody.querySelectorAll('.credential-btn-delete').forEach(btn => {
                    btn.addEventListener('click', function() {
                        const tr = this.closest('tr');
                        if (tr) self.deleteCredentialInSelect(tr.dataset.credId, tr.dataset.credName);
                    });
                });
            } catch (error) {
                tbody.innerHTML = `<tr><td colspan="4" class="text-center text-danger py-4">加载失败: ${error.message || ''}</td></tr>`;
            }
        },

        // 选择凭据并关闭弹框
        selectCredential: function(id, name, username) {
            document.getElementById('dsCredential').value = id;
            document.getElementById('dsCredentialName').value = `${name} (${username})`;
            this.closeCredentialSelectModal();
        },
        
        // 显示添加凭据表单
        showAddCredentialForm: function() {
            document.getElementById('credentialFormModalTitle').textContent = '新建凭据';
            document.getElementById('credentialFormId').value = '';
            document.getElementById('credentialFormInModal').reset();
            document.getElementById('credFormPassword').required = true;
            document.getElementById('credFormPasswordRequired').innerHTML = '*';
            document.getElementById('credentialFormModal').style.display = 'flex';
        },

        // 显示编辑凭据表单
        showEditCredentialForm: async function(id) {
            try {
                const result = await HttpUtils.get('/api/v1/credentials/' + id);
                if (result.code !== 200 || !result.data) {
                    Toast.error('加载凭据失败');
                    return;
                }
                const c = result.data;
                document.getElementById('credentialFormModalTitle').textContent = '编辑凭据';
                document.getElementById('credentialFormId').value = c.id;
                document.getElementById('credFormName').value = c.name || '';
                document.getElementById('credFormUsername').value = c.username || '';
                document.getElementById('credFormPassword').value = '';
                document.getElementById('credFormPassword').placeholder = '留空表示不修改';
                document.getElementById('credFormPassword').required = false;
                document.getElementById('credFormPasswordRequired').innerHTML = '';
                document.getElementById('credFormDescription').value = c.description || '';
                document.getElementById('credentialFormModal').style.display = 'flex';
            } catch (error) {
                Toast.error('加载凭据失败: ' + (error.message || ''));
            }
        },

        // 关闭凭据表单弹框
        closeCredentialFormModal: function() {
            document.getElementById('credentialFormModal').style.display = 'none';
        },

        // 保存凭据
        saveCredentialForm: async function(e) {
            e.preventDefault();
            const id = document.getElementById('credentialFormId').value;
            const data = {
                name: document.getElementById('credFormName').value.trim(),
                username: document.getElementById('credFormUsername').value.trim(),
                password: document.getElementById('credFormPassword').value,
                description: document.getElementById('credFormDescription').value.trim()
            };
            if (id && !data.password) delete data.password;
            
            try {
                if (id) {
                    await HttpUtils.put('/api/v1/credentials/' + id, data);
                    Toast.success('更新成功');
                } else {
                    await HttpUtils.post('/api/v1/credentials', data);
                    Toast.success('创建成功');
                }
                this.closeCredentialFormModal();
                await this.loadCredentialSelectList();
            } catch (error) {
                Toast.error(error.message || '保存失败');
            }
        },

        // 删除凭据
        deleteCredentialInSelect: function(id, name) {
            const self = this;
            const safeName = this.escapeHtml(name || id || '');
            
            if (typeof Modal !== 'undefined' && Modal.confirm) {
                Modal.confirm(`确定要删除凭据"${safeName}"吗？<br><span class="text-danger small">被数据源使用的凭据无法删除。</span>`, function() {
                    self.doDeleteCredentialInSelect(id);
                });
            } else if (confirm(`确定要删除凭据"${name || id}"吗？`)) {
                this.doDeleteCredentialInSelect(id);
            }
        },

        doDeleteCredentialInSelect: async function(id) {
            try {
                await HttpUtils.delete('/api/v1/credentials/' + id);
                Toast.success('删除成功');
                await this.loadCredentialSelectList();
            } catch (error) {
                Toast.error(error.message || '删除失败');
            }
        },
        
        // HTML转义工具方法
        escapeHtml: function(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }
    };
})();
