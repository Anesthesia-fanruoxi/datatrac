// 任务配置向导 - 步骤2：选择表（主模块）
(function() {
    'use strict';
    
    window.TaskWizardStep2 = {
        allDatabases: [],
        
        // 渲染步骤内容
        render: async function(container, taskData) {
            container.innerHTML = '<div class="text-center"><div class="spinner-border"></div><div>加载数据库和表...</div></div>';
            
            try {
                // 调用新接口：一次性获取所有数据库和表
                const result = await HttpUtils.get(`/api/v1/datasources/${taskData.source_id}/database-tables`);
                
                if (result.code !== 200) {
                    container.innerHTML = '<div class="alert alert-danger">加载数据库和表失败: ' + result.message + '</div>';
                    return;
                }
                
                // 过滤空库（表数量为0的数据库）
                this.allDatabases = (result.data || []).filter(db => db.tables && db.tables.length > 0);
                
                if (this.allDatabases.length === 0) {
                    container.innerHTML = '<div class="alert alert-warning">该数据源没有可用的数据库或表</div>';
                    return;
                }
                
                // 初始化映射管理器
                if (taskData.selected_databases && Array.isArray(taskData.selected_databases)) {
                    // 从已有配置加载
                    TaskWizardStep2Mapping.loadFromConfig(taskData.selected_databases);
                } else {
                    // 清空映射
                    TaskWizardStep2Mapping.mappings = {};
                }
                
                container.innerHTML = `
                    <div class="transfer-container">
                        <!-- 左侧：源数据库树 -->
                        <div class="transfer-panel">
                            <div class="transfer-panel-header">
                                <span><i class="bi bi-database me-2"></i>源数据库</span>
                                <div>
                                    <button class="btn btn-sm btn-link" onclick="TaskWizardStep2.selectAllLeft()">全选</button>
                                    <button class="btn btn-sm btn-link" onclick="TaskWizardStep2.deselectAllLeft()">取消</button>
                                </div>
                            </div>
                            <div class="transfer-panel-body" id="leftPanel">
                                ${this.renderLeftTree()}
                            </div>
                        </div>
                        
                        <!-- 中间：穿梭按钮 -->
                        <div class="transfer-buttons">
                            <button class="btn btn-primary btn-sm mb-2" onclick="TaskWizardStep2.moveToRight()" title="添加选中的表">
                                <i class="bi bi-chevron-right"></i>
                            </button>
                            <button class="btn btn-outline-secondary btn-sm" onclick="TaskWizardStep2.moveToLeft()" title="移除选中的表">
                                <i class="bi bi-chevron-left"></i>
                            </button>
                        </div>
                        
                        <!-- 右侧：已选择的表（带映射） -->
                        <div class="transfer-panel">
                            <div class="transfer-panel-header">
                                <span><i class="bi bi-check-square me-2"></i>已选择的表</span>
                                <div>
                                    <button class="btn btn-sm btn-primary" onclick="TaskWizardStep2.showBatchEditDatabase()" 
                                        style="background: #3b82f6; border: none;" title="批量修改库名">
                                        <i class="bi bi-database"></i>
                                    </button>
                                    <button class="btn btn-sm btn-warning ms-1" onclick="TaskWizardStep2.showBatchEditTable()" 
                                        style="background: #fbbf24; border: none;" title="批量修改表名">
                                        <i class="bi bi-table"></i>
                                    </button>
                                    <button class="btn btn-sm btn-danger ms-1" onclick="TaskWizardStep2.clearAll()" 
                                        style="background: #ef4444; border: none;" title="清空">
                                        <i class="bi bi-trash"></i>
                                    </button>
                                </div>
                            </div>
                            <div class="transfer-panel-body" id="rightPanel">
                                ${this.renderRightTree()}
                            </div>
                        </div>
                    </div>
                `;
                
                // 保存数据
                taskData.databases = this.allDatabases;
                
                // 绑定事件
                this.bindEvents();
                
            } catch (error) {
                container.innerHTML = '<div class="alert alert-danger">加载失败: ' + error.message + '</div>';
            }
        },
        
        // 渲染左侧树
        renderLeftTree: function() {
            const selectedTables = TaskWizardStep2Mapping.getSelectedTables();
            
            return this.allDatabases.map(db => {
                const tables = db.tables || [];
                const dbKey = db.database;
                const selectedDbTables = selectedTables[dbKey] || [];
                
                return `
                    <div class="tree-node">
                        <div class="tree-node-header" onclick="TaskWizardStep2.toggleLeftDatabase('${dbKey}')">
                            <i class="bi bi-chevron-right me-1 tree-chevron" id="left-chevron-${dbKey}"></i>
                            <input type="checkbox" class="form-check-input me-2 left-db-checkbox" data-db="${dbKey}" 
                                ${selectedDbTables.length === tables.length && tables.length > 0 ? 'checked' : ''}
                                onclick="event.stopPropagation()">
                            <i class="bi bi-database me-2"></i>
                            <strong>${db.database}</strong>
                            <span class="text-muted ms-2">(${tables.length})</span>
                        </div>
                        <div class="tree-node-children" id="left-tables-${dbKey}" style="display:none;">
                            ${tables.map(table => `
                                <div class="tree-node-item">
                                    <input type="checkbox" class="form-check-input me-2 left-table-checkbox" 
                                        data-db="${dbKey}" data-table="${table}"
                                        ${selectedDbTables.includes(table) ? 'checked' : ''}>
                                    <i class="bi bi-table me-2"></i>
                                    ${table}
                                </div>
                            `).join('')}
                        </div>
                    </div>
                `;
            }).join('');
        },
        
        // 渲染右侧树（带映射）
        renderRightTree: function() {
            const mappings = TaskWizardStep2Mapping.mappings;
            const dbNames = Object.keys(mappings);
            
            if (dbNames.length === 0) {
                return '<div class="empty-state"><i class="bi bi-inbox"></i><p>暂无选择</p></div>';
            }
            
            return dbNames.map(dbKey => {
                const mapping = mappings[dbKey];
                const tables = mapping.tables || [];
                const isDbModified = mapping.is_database_modified;
                
                return `
                    <div class="tree-node">
                        <div class="tree-node-header" onclick="TaskWizardStep2.toggleRightDatabase('${dbKey}')">
                            <i class="bi bi-chevron-right me-1 tree-chevron" id="right-chevron-${dbKey}"></i>
                            <input type="checkbox" class="form-check-input me-2 right-db-checkbox" data-db="${dbKey}" onclick="event.stopPropagation()">
                            <i class="bi bi-database me-2"></i>
                            <span class="mapping-name">
                                ${isDbModified ? `<span style="color: #28a745; font-weight: 500;">${mapping.target_database}</span> <span class="text-muted small">(原: ${mapping.source_database})</span>` : mapping.source_database}
                            </span>
                            <i class="bi bi-pencil ms-2 edit-icon" onclick="event.stopPropagation(); TaskWizardStep2.editDatabaseName('${dbKey}')" title="编辑库名"></i>
                            <span class="text-muted ms-2">(${tables.length})</span>
                        </div>
                        <div class="tree-node-children" id="right-tables-${dbKey}" style="display:none;">
                            ${tables.map(table => `
                                <div class="tree-node-item">
                                    <input type="checkbox" class="form-check-input me-2 right-table-checkbox" 
                                        data-db="${dbKey}" data-table="${table.source_table}">
                                    <i class="bi bi-table me-2"></i>
                                    <span class="mapping-name">
                                        ${table.is_modified ? `<span style="color: #28a745; font-weight: 500;">${table.target_table}</span> <span class="text-muted small">(原: ${table.source_table})</span>` : table.source_table}
                                    </span>
                                    <i class="bi bi-pencil ms-2 edit-icon" onclick="TaskWizardStep2.editTableName('${dbKey}', '${table.source_table}')" title="编辑表名"></i>
                                </div>
                            `).join('')}
                        </div>
                    </div>
                `;
            }).join('');
        },
        
        // 绑定事件
        bindEvents: function() {
            // 左侧数据库复选框
            document.querySelectorAll('.left-db-checkbox').forEach(checkbox => {
                checkbox.addEventListener('change', function() {
                    const db = this.getAttribute('data-db');
                    const checked = this.checked;
                    document.querySelectorAll(`.left-table-checkbox[data-db="${db}"]`).forEach(cb => {
                        cb.checked = checked;
                    });
                });
            });
            
            // 左侧表复选框
            document.querySelectorAll('.left-table-checkbox').forEach(checkbox => {
                checkbox.addEventListener('change', function() {
                    const db = this.getAttribute('data-db');
                    const allTables = document.querySelectorAll(`.left-table-checkbox[data-db="${db}"]`);
                    const checkedTables = document.querySelectorAll(`.left-table-checkbox[data-db="${db}"]:checked`);
                    const dbCheckbox = document.querySelector(`.left-db-checkbox[data-db="${db}"]`);
                    if (dbCheckbox) {
                        dbCheckbox.checked = allTables.length === checkedTables.length;
                    }
                });
            });
            
            // 右侧数据库复选框
            document.querySelectorAll('.right-db-checkbox').forEach(checkbox => {
                checkbox.addEventListener('change', function() {
                    const db = this.getAttribute('data-db');
                    const checked = this.checked;
                    document.querySelectorAll(`.right-table-checkbox[data-db="${db}"]`).forEach(cb => {
                        cb.checked = checked;
                    });
                });
            });
            
            // 右侧表复选框
            document.querySelectorAll('.right-table-checkbox').forEach(checkbox => {
                checkbox.addEventListener('change', function() {
                    const db = this.getAttribute('data-db');
                    const allTables = document.querySelectorAll(`.right-table-checkbox[data-db="${db}"]`);
                    const checkedTables = document.querySelectorAll(`.right-table-checkbox[data-db="${db}"]:checked`);
                    const dbCheckbox = document.querySelector(`.right-db-checkbox[data-db="${db}"]`);
                    if (dbCheckbox) {
                        dbCheckbox.checked = allTables.length === checkedTables.length;
                    }
                });
            });
        },
        
        // 切换左侧数据库展开/折叠
        toggleLeftDatabase: function(dbKey) {
            const tables = document.getElementById(`left-tables-${dbKey}`);
            const chevron = document.getElementById(`left-chevron-${dbKey}`);
            if (tables.style.display === 'none') {
                tables.style.display = 'block';
                chevron.classList.add('rotated');
            } else {
                tables.style.display = 'none';
                chevron.classList.remove('rotated');
            }
        },
        
        // 切换右侧数据库展开/折叠
        toggleRightDatabase: function(dbKey) {
            const tables = document.getElementById(`right-tables-${dbKey}`);
            const chevron = document.getElementById(`right-chevron-${dbKey}`);
            if (tables && chevron) {
                if (tables.style.display === 'none') {
                    tables.style.display = 'block';
                    chevron.classList.add('rotated');
                } else {
                    tables.style.display = 'none';
                    chevron.classList.remove('rotated');
                }
            }
        },
        
        // 左侧全选
        selectAllLeft: function() {
            document.querySelectorAll('.left-table-checkbox, .left-db-checkbox').forEach(cb => cb.checked = true);
        },
        
        // 左侧取消全选
        deselectAllLeft: function() {
            document.querySelectorAll('.left-table-checkbox, .left-db-checkbox').forEach(cb => cb.checked = false);
        },
        
        // 移动到右侧
        moveToRight: function() {
            const selectedTables = {};
            
            document.querySelectorAll('.left-table-checkbox:checked').forEach(checkbox => {
                const db = checkbox.getAttribute('data-db');
                const table = checkbox.getAttribute('data-table');
                
                if (!selectedTables[db]) {
                    selectedTables[db] = [];
                }
                selectedTables[db].push(table);
                checkbox.checked = false;
            });
            
            // 添加到映射
            TaskWizardStep2Mapping.addFromSelectedTables(selectedTables);
            
            // 取消数据库复选框
            document.querySelectorAll('.left-db-checkbox').forEach(cb => cb.checked = false);
            
            // 刷新两侧
            this.refreshBoth();
        },
        
        // 移动到左侧（移除）
        moveToLeft: function() {
            document.querySelectorAll('.right-table-checkbox:checked').forEach(checkbox => {
                const db = checkbox.getAttribute('data-db');
                const table = checkbox.getAttribute('data-table');
                TaskWizardStep2Mapping.removeTable(db, table);
            });
            
            // 刷新两侧
            this.refreshBoth();
        },
        
        // 清空所有
        clearAll: function() {
            Modal.confirm('确定要清空所有选择吗？', () => {
                TaskWizardStep2Mapping.mappings = {};
                this.refreshBoth();
            });
        },
        
        // 编辑数据库名
        editDatabaseName: function(dbKey) {
            const mapping = TaskWizardStep2Mapping.mappings[dbKey];
            if (!mapping) return;
            
            Modal.prompt(
                '编辑数据库名',
                '请输入目标数据库名:',
                mapping.target_database,
                (newName) => {
                    TaskWizardStep2Mapping.updateDatabaseMapping(dbKey, newName);
                    this.refreshRight();
                }
            );
        },
        
        // 编辑表名
        editTableName: function(dbKey, tableName) {
            const mapping = TaskWizardStep2Mapping.mappings[dbKey];
            if (!mapping) return;
            
            const table = mapping.tables.find(t => t.source_table === tableName);
            if (!table) return;
            
            Modal.prompt(
                '编辑表名',
                '请输入目标表名:',
                table.target_table,
                (newName) => {
                    TaskWizardStep2Mapping.updateTableMapping(dbKey, tableName, newName);
                    this.refreshRight();
                }
            );
        },
        
        // 重置数据库映射
        resetDatabaseMapping: function(dbKey) {
            TaskWizardStep2Mapping.resetMapping(dbKey);
            this.refreshRight();
        },
        
        // 重置表映射
        resetTableMapping: function(dbKey, tableName) {
            TaskWizardStep2Mapping.resetMapping(dbKey, tableName);
            this.refreshRight();
        },
        
        // 显示批量编辑库名对话框
        showBatchEditDatabase: function() {
            const mappings = TaskWizardStep2Mapping.mappings;
            const dbNames = Object.keys(mappings);
            
            if (dbNames.length === 0) {
                Toast.warning('请先选择数据库');
                return;
            }
            
            this.showBatchEditModal('database', dbNames);
        },
        
        // 显示批量编辑表名对话框
        showBatchEditTable: function() {
            const mappings = TaskWizardStep2Mapping.mappings;
            const allTables = {};
            
            Object.keys(mappings).forEach(db => {
                const tables = mappings[db].tables.map(t => t.source_table);
                if (tables.length > 0) {
                    allTables[db] = tables;
                }
            });
            
            const totalTables = Object.values(allTables).reduce((sum, tables) => sum + tables.length, 0);
            
            if (totalTables === 0) {
                Toast.warning('请先选择表');
                return;
            }
            
            this.showBatchEditModal('table', allTables);
        },
        
        // 显示批量编辑模态框
        showBatchEditModal: function(type, selected) {
            const isDatabase = type === 'database';
            const count = isDatabase ? selected.length : Object.values(selected).reduce((sum, tables) => sum + tables.length, 0);
            const title = isDatabase ? '批量编辑库名' : '批量编辑表名';
            const icon = isDatabase ? 'database' : 'table';
            const target = isDatabase ? '数据库' : '表';
            
            const modalHtml = `
            <div id="batchEditModal" style="display:block; position:fixed; top:0; left:0; width:100%; height:100%; background:rgba(0,0,0,0.6); z-index:2000;">
                <div style="position:absolute; top:50%; left:50%; transform:translate(-50%,-50%); background:white; padding:0; border-radius:15px; width:600px; box-shadow: 0 20px 60px rgba(0,0,0,0.3);">
                    <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 20px; border-radius: 15px 15px 0 0;">
                        <h5 style="color: white; margin: 0;"><i class="bi bi-${icon} me-2"></i>${title}</h5>
                        <div style="color: rgba(255,255,255,0.9); font-size: 14px; margin-top: 5px;">
                            将对所有 ${count} 个${target}进行批量操作
                        </div>
                    </div>
                    <div style="padding: 30px;">
                        <div class="mb-4">
                            <label class="form-label fw-bold mb-3">操作类型</label>
                            <div class="d-flex gap-2 mb-3">
                                <div class="form-check-card flex-fill">
                                    <input class="form-check-input" type="radio" name="batchEditType" id="type-add-prefix" value="add-prefix" checked>
                                    <label class="form-check-label d-flex align-items-center" for="type-add-prefix">
                                        <i class="bi bi-plus-circle text-success me-2"></i>
                                        <span>添加前缀</span>
                                    </label>
                                </div>
                                <div class="form-check-card flex-fill">
                                    <input class="form-check-input" type="radio" name="batchEditType" id="type-add-suffix" value="add-suffix">
                                    <label class="form-check-label d-flex align-items-center" for="type-add-suffix">
                                        <i class="bi bi-plus-circle text-success me-2"></i>
                                        <span>添加后缀</span>
                                    </label>
                                </div>
                                <div class="form-check-card flex-fill">
                                    <input class="form-check-input" type="radio" name="batchEditType" id="type-replace-prefix" value="replace-prefix">
                                    <label class="form-check-label d-flex align-items-center" for="type-replace-prefix">
                                        <i class="bi bi-arrow-repeat text-primary me-2"></i>
                                        <span>替换前缀</span>
                                    </label>
                                </div>
                                <div class="form-check-card flex-fill">
                                    <input class="form-check-input" type="radio" name="batchEditType" id="type-replace-suffix" value="replace-suffix">
                                    <label class="form-check-label d-flex align-items-center" for="type-replace-suffix">
                                        <i class="bi bi-arrow-repeat text-primary me-2"></i>
                                        <span>替换后缀</span>
                                    </label>
                                </div>
                            </div>
                            <div class="text-muted" style="font-size: 13px; line-height: 1.8;">
                                <div><strong>操作说明：</strong>将对所有 ${count} 个${target}进行批量操作</div>
                                <div id="batchEditExample"><strong>示例：</strong>添加前缀"new_"后，"users"变为"new_users"</div>
                            </div>
                        </div>
                        
                        <div id="batchEditInputs">
                            <div class="mb-3">
                                <label class="form-label fw-bold">前缀</label>
                                <input type="text" class="form-control" id="batchEditValue" placeholder="请输入前缀">
                                <div class="form-text">将为所有${target}添加此前缀</div>
                            </div>
                        </div>
                        
                        <div class="d-flex justify-content-end gap-2 mt-4">
                            <button type="button" class="btn btn-secondary" onclick="TaskWizardStep2.closeBatchEditModal()">取消</button>
                            <button type="button" class="btn btn-primary" onclick="TaskWizardStep2.applyBatchEdit()">
                                <i class="bi bi-check-circle"></i> 应用
                            </button>
                        </div>
                    </div>
                </div>
            </div>
            `;
            
            document.body.insertAdjacentHTML('beforeend', modalHtml);
            
            // 保存选中的数据
            this.batchEditType = type;
            this.batchEditSelected = selected;
            
            // 绑定单选按钮切换事件
            document.querySelectorAll('input[name="batchEditType"]').forEach(radio => {
                radio.addEventListener('change', function() {
                    TaskWizardStep2.updateBatchEditInputs(this.value);
                    TaskWizardStep2.updateBatchEditExample(this.value);
                });
            });
        },
        
        // 更新批量编辑示例
        updateBatchEditExample: function(type) {
            const exampleDiv = document.getElementById('batchEditExample');
            if (!exampleDiv) return;
            
            let exampleText = '';
            switch(type) {
                case 'add-prefix':
                    exampleText = '添加前缀"new_"后，"users"变为"new_users"';
                    break;
                case 'add-suffix':
                    exampleText = '添加后缀"_bak"后，"users"变为"users_bak"';
                    break;
                case 'replace-prefix':
                    exampleText = '替换前缀"old_"为"new_"后，"old_users"变为"new_users"';
                    break;
                case 'replace-suffix':
                    exampleText = '替换后缀"_old"为"_new"后，"users_old"变为"users_new"';
                    break;
            }
            
            exampleDiv.innerHTML = `<strong>示例：</strong>${exampleText}`;
        },
        
        // 更新批量编辑输入框
        updateBatchEditInputs: function(type) {
            const container = document.getElementById('batchEditInputs');
            const isDatabase = this.batchEditType === 'database';
            const target = isDatabase ? '数据库' : '表';
            
            if (type === 'add-prefix') {
                container.innerHTML = `
                    <div class="mb-3">
                        <label class="form-label">前缀</label>
                        <input type="text" class="form-control" id="batchEditValue" placeholder="请输入前缀">
                        <div class="form-text">将为所有${target}添加此前缀</div>
                    </div>
                `;
            } else if (type === 'add-suffix') {
                container.innerHTML = `
                    <div class="mb-3">
                        <label class="form-label">后缀</label>
                        <input type="text" class="form-control" id="batchEditValue" placeholder="请输入后缀">
                        <div class="form-text">将为所有${target}添加此后缀</div>
                    </div>
                `;
            } else if (type === 'replace-prefix') {
                container.innerHTML = `
                    <div class="mb-3">
                        <label class="form-label">旧前缀</label>
                        <input type="text" class="form-control" id="batchEditOldValue" placeholder="请输入要替换的前缀">
                        <div class="form-text">只替换以此前缀开头的${target}</div>
                    </div>
                    <div class="mb-3">
                        <label class="form-label">新前缀</label>
                        <input type="text" class="form-control" id="batchEditNewValue" placeholder="请输入新前缀（可为空）">
                    </div>
                `;
            } else if (type === 'replace-suffix') {
                container.innerHTML = `
                    <div class="mb-3">
                        <label class="form-label">旧后缀</label>
                        <input type="text" class="form-control" id="batchEditOldValue" placeholder="请输入要替换的后缀">
                        <div class="form-text">只替换以此后缀结尾的${target}</div>
                    </div>
                    <div class="mb-3">
                        <label class="form-label">新后缀</label>
                        <input type="text" class="form-control" id="batchEditNewValue" placeholder="请输入新后缀（可为空）">
                    </div>
                `;
            }
        },
        
        // 应用批量编辑
        applyBatchEdit: function() {
            const editType = document.querySelector('input[name="batchEditType"]:checked').value;
            const selected = this.batchEditSelected;
            const isDatabase = this.batchEditType === 'database';
            
            if (isDatabase) {
                // 批量编辑数据库名
                if (editType === 'add-prefix') {
                    const prefix = document.getElementById('batchEditValue').value.trim();
                    if (!prefix) {
                        Toast.warning('请输入前缀');
                        return;
                    }
                    selected.forEach(db => {
                        const mapping = TaskWizardStep2Mapping.mappings[db];
                        if (mapping) {
                            TaskWizardStep2Mapping.updateDatabaseMapping(db, prefix + mapping.source_database);
                        }
                    });
                } else if (editType === 'add-suffix') {
                    const suffix = document.getElementById('batchEditValue').value.trim();
                    if (!suffix) {
                        Toast.warning('请输入后缀');
                        return;
                    }
                    selected.forEach(db => {
                        const mapping = TaskWizardStep2Mapping.mappings[db];
                        if (mapping) {
                            TaskWizardStep2Mapping.updateDatabaseMapping(db, mapping.source_database + suffix);
                        }
                    });
                } else if (editType === 'replace-prefix') {
                    const oldPrefix = document.getElementById('batchEditOldValue').value.trim();
                    const newPrefix = document.getElementById('batchEditNewValue').value.trim();
                    if (!oldPrefix) {
                        Toast.warning('请输入旧前缀');
                        return;
                    }
                    selected.forEach(db => {
                        const mapping = TaskWizardStep2Mapping.mappings[db];
                        if (mapping && mapping.source_database.startsWith(oldPrefix)) {
                            TaskWizardStep2Mapping.updateDatabaseMapping(db, newPrefix + mapping.source_database.substring(oldPrefix.length));
                        }
                    });
                } else if (editType === 'replace-suffix') {
                    const oldSuffix = document.getElementById('batchEditOldValue').value.trim();
                    const newSuffix = document.getElementById('batchEditNewValue').value.trim();
                    if (!oldSuffix) {
                        Toast.warning('请输入旧后缀');
                        return;
                    }
                    selected.forEach(db => {
                        const mapping = TaskWizardStep2Mapping.mappings[db];
                        if (mapping && mapping.source_database.endsWith(oldSuffix)) {
                            TaskWizardStep2Mapping.updateDatabaseMapping(db, mapping.source_database.substring(0, mapping.source_database.length - oldSuffix.length) + newSuffix);
                        }
                    });
                }
            } else {
                // 批量编辑表名
                if (editType === 'add-prefix') {
                    const prefix = document.getElementById('batchEditValue').value.trim();
                    if (!prefix) {
                        Toast.warning('请输入前缀');
                        return;
                    }
                    Object.keys(selected).forEach(db => {
                        TaskWizardStep2Mapping.batchAddPrefix(db, selected[db], prefix);
                    });
                } else if (editType === 'add-suffix') {
                    const suffix = document.getElementById('batchEditValue').value.trim();
                    if (!suffix) {
                        Toast.warning('请输入后缀');
                        return;
                    }
                    Object.keys(selected).forEach(db => {
                        TaskWizardStep2Mapping.batchAddSuffix(db, selected[db], suffix);
                    });
                } else if (editType === 'replace-prefix') {
                    const oldPrefix = document.getElementById('batchEditOldValue').value.trim();
                    const newPrefix = document.getElementById('batchEditNewValue').value.trim();
                    if (!oldPrefix) {
                        Toast.warning('请输入旧前缀');
                        return;
                    }
                    Object.keys(selected).forEach(db => {
                        TaskWizardStep2Mapping.batchReplacePrefix(db, selected[db], oldPrefix, newPrefix);
                    });
                } else if (editType === 'replace-suffix') {
                    const oldSuffix = document.getElementById('batchEditOldValue').value.trim();
                    const newSuffix = document.getElementById('batchEditNewValue').value.trim();
                    if (!oldSuffix) {
                        Toast.warning('请输入旧后缀');
                        return;
                    }
                    Object.keys(selected).forEach(db => {
                        TaskWizardStep2Mapping.batchReplaceSuffix(db, selected[db], oldSuffix, newSuffix);
                    });
                }
            }
            
            this.closeBatchEditModal();
            this.refreshRight();
            Toast.success('批量编辑成功');
        },
        
        // 关闭批量编辑模态框
        closeBatchEditModal: function() {
            const modal = document.getElementById('batchEditModal');
            if (modal) {
                modal.remove();
            }
        },
        
        // 刷新右侧面板
        refreshRight: function() {
            const rightPanel = document.getElementById('rightPanel');
            if (rightPanel) {
                // 保存当前展开状态
                const expandedDbs = [];
                document.querySelectorAll('.tree-chevron.rotated').forEach(chevron => {
                    const id = chevron.id;
                    if (id.startsWith('right-chevron-')) {
                        expandedDbs.push(id.replace('right-chevron-', ''));
                    }
                });
                
                // 重新渲染
                rightPanel.innerHTML = this.renderRightTree();
                this.bindEvents();
                
                // 恢复展开状态
                expandedDbs.forEach(dbKey => {
                    const tables = document.getElementById(`right-tables-${dbKey}`);
                    const chevron = document.getElementById(`right-chevron-${dbKey}`);
                    if (tables && chevron) {
                        tables.style.display = 'block';
                        chevron.classList.add('rotated');
                    }
                });
            }
        },
        
        // 刷新两侧面板
        refreshBoth: function() {
            const leftPanel = document.getElementById('leftPanel');
            const rightPanel = document.getElementById('rightPanel');
            
            // 保存右侧展开状态
            const expandedDbs = [];
            document.querySelectorAll('.tree-chevron.rotated').forEach(chevron => {
                const id = chevron.id;
                if (id.startsWith('right-chevron-')) {
                    expandedDbs.push(id.replace('right-chevron-', ''));
                }
            });
            
            if (leftPanel) {
                leftPanel.innerHTML = this.renderLeftTree();
            }
            if (rightPanel) {
                rightPanel.innerHTML = this.renderRightTree();
            }
            
            this.bindEvents();
            
            // 恢复右侧展开状态
            expandedDbs.forEach(dbKey => {
                const tables = document.getElementById(`right-tables-${dbKey}`);
                const chevron = document.getElementById(`right-chevron-${dbKey}`);
                if (tables && chevron) {
                    tables.style.display = 'block';
                    chevron.classList.add('rotated');
                }
            });
        },
        
        // 验证步骤
        validate: async function(taskData) {
            const mappings = TaskWizardStep2Mapping.mappings;
            
            if (!mappings || Object.keys(mappings).length === 0) {
                Toast.warning('请至少选择一张表');
                return false;
            }
            
            // 转换为配置格式
            taskData.selected_databases = TaskWizardStep2Mapping.toConfig();
            taskData.selected_tables = TaskWizardStep2Mapping.getSelectedTables();
            
            return true;
        }
    };
})();
