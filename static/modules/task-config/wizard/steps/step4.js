// 任务配置向导 - 步骤4：确认配置
(function() {
    'use strict';
    
    window.TaskWizardStep4 = {
        // 渲染步骤内容
        render: function(container, taskData) {
            const task = taskData.task;
            const syncConfig = taskData.sync_config || {};
            const selectedDatabases = taskData.selected_databases || [];
            const targetIDs = taskData.target_ids || [];
            
            // 统计信息
            const totalTables = selectedDatabases.reduce((sum, db) => sum + (db.tables ? db.tables.length : 0), 0);
            const totalDatabases = selectedDatabases.length;
            
            // 统计修改数量
            const modifiedDatabases = selectedDatabases.filter(db => db.is_database_modified).length;
            const modifiedTables = selectedDatabases.reduce((sum, db) => {
                const tables = db.tables || [];
                return sum + tables.filter(t => t.is_modified).length;
            }, 0);
            
            // 获取同步模式文本
            const syncModeText = this.getSyncModeText(syncConfig.sync_mode);
            
            container.innerHTML = `
                <div>
                    <div class="row">
                        <div class="col-md-6">
                            <div class="config-section">
                                <h6><i class="bi bi-info-circle me-2"></i>基本信息</h6>
                                <table class="table table-sm">
                                    <tr><td>任务名称</td><td><strong>${task.name}</strong></td></tr>
                                    <tr><td>同步方向</td><td>${task.source_type} → ${task.target_type}</td></tr>
                                    <tr><td>目标源数量</td><td><strong>${targetIDs.length} 个</strong></td></tr>
                                    <tr>
                                        <td>选中数据库</td>
                                        <td>
                                            ${totalDatabases} 个 
                                            ${modifiedDatabases > 0 ? `<span class="text-success" style="cursor: pointer; text-decoration: underline;" onclick="TaskWizardStep4.showModifiedDatabases()">(修改 ${modifiedDatabases} 个)</span>` : ''}
                                        </td>
                                    </tr>
                                    <tr>
                                        <td>选中表</td>
                                        <td>
                                            ${totalTables} 张 
                                            ${modifiedTables > 0 ? `<span class="text-success" style="cursor: pointer; text-decoration: underline;" onclick="TaskWizardStep4.showModifiedTables()">(修改 ${modifiedTables} 张)</span>` : ''}
                                        </td>
                                    </tr>
                                </table>
                            </div>
                            
                            <div class="config-section mt-3">
                                <h6><i class="bi bi-gear me-2"></i>同步配置</h6>
                                <table class="table table-sm">
                                    <tr><td>同步模式</td><td><strong>${syncModeText}</strong></td></tr>
                                    ${syncConfig.sync_mode !== 'structure' ? `
                                    <tr><td>错误策略</td><td>${syncConfig.error_strategy === 'skip' ? '跳过错误' : '遇错暂停'}</td></tr>
                                    <tr><td>表存在策略</td><td>${this.getTableStrategyText(syncConfig.table_exists_strategy)}</td></tr>
                                    ` : ''}
                                    <tr><td colspan="2" class="text-muted"><small><i class="bi bi-cpu me-1"></i>${syncConfig.sync_mode === 'structure' ? '仅对比和同步表结构，不同步数据' : '批次大小和线程数将根据系统资源自动优化'}</small></td></tr>
                                </table>
                            </div>
                        </div>
                        
                        <div class="col-md-6">
                            <div class="config-section">
                                <h6><i class="bi bi-table me-2"></i>选中的表</h6>
                                <div style="max-height: 400px; overflow-y: auto;">
                                    ${this.renderSelectedTables(selectedDatabases)}
                                </div>
                            </div>
                        </div>
                    </div>
                    
                    <div class="alert alert-success mt-3">
                        <i class="bi bi-check-circle me-2"></i>
                        配置完成！点击"完成配置"按钮保存配置并返回任务列表。
                    </div>
                </div>
            `;
        },
        
        // 获取同步模式文本
        getSyncModeText: function(syncMode) {
            const texts = {
                'full': '全量同步',
                'incremental': '增量同步',
                'structure': '只同步表结构'
            };
            return texts[syncMode] || syncMode;
        },
        
        // 获取表存在策略文本
        getTableStrategyText: function(strategy) {
            const texts = {
                'drop': '删除重建',
                'truncate': '清空数据',
                'backup': '备份后重建'
            };
            return texts[strategy] || strategy;
        },
        
        // 渲染选中的表（树状结构）
        renderSelectedTables: function(selectedDatabases) {
            if (!selectedDatabases || selectedDatabases.length === 0) {
                return '<div class="text-muted">未选择表</div>';
            }
            
            return selectedDatabases.map(dbMapping => {
                const isDbModified = dbMapping.is_database_modified;
                const tables = dbMapping.tables || [];
                const dbKey = dbMapping.database || dbMapping.source_database;
                const targetDb = dbMapping.database || dbMapping.target_database;
                const sourceDb = dbMapping.source_database || dbMapping.database;
                
                return `
                    <div class="tree-node mb-2">
                        <div class="tree-node-header" onclick="TaskWizardStep4.toggleDatabase('${dbKey}')" style="cursor: pointer; padding: 8px; border-radius: 6px; background: #f8f9fa;">
                            <i class="bi bi-chevron-right me-1 tree-chevron" id="step4-chevron-${dbKey}" style="transition: transform 0.2s; font-size: 12px;"></i>
                            <i class="bi bi-database me-2 text-primary"></i>
                            <strong>
                                ${isDbModified ? `<span style="color: #28a745;">${targetDb}</span> <span class="text-muted small">(原: ${sourceDb})</span>` : targetDb}
                            </strong>
                            <span class="text-muted ms-2">(${tables.length})</span>
                        </div>
                        <div class="tree-node-children ms-3" id="step4-tables-${dbKey}" style="display:none;">
                            ${tables.map(table => `
                                <div class="tree-node-item" style="padding: 6px 10px; border-radius: 4px; font-size: 13px;">
                                    <i class="bi bi-table me-2 text-success"></i>
                                    ${table.is_modified ? `<span style="color: #28a745;">${table.target_table}</span> <span class="text-muted small">(原: ${table.source_table})</span>` : table.source_table}
                                </div>
                            `).join('')}
                        </div>
                    </div>
                `;
            }).join('');
        },
        
        // 切换数据库展开/折叠
        toggleDatabase: function(dbKey) {
            const tables = document.getElementById(`step4-tables-${dbKey}`);
            const chevron = document.getElementById(`step4-chevron-${dbKey}`);
            if (tables && chevron) {
                if (tables.style.display === 'none') {
                    tables.style.display = 'block';
                    chevron.style.transform = 'rotate(90deg)';
                } else {
                    tables.style.display = 'none';
                    chevron.style.transform = 'rotate(0deg)';
                }
            }
        },
        
        // 显示修改的数据库
        showModifiedDatabases: function() {
            const taskData = window.TaskWizard.taskData;
            const selectedDatabases = taskData.selected_databases || [];
            const modifiedDbs = selectedDatabases.filter(db => db.is_database_modified);
            
            if (modifiedDbs.length === 0) {
                Toast.info('没有修改的数据库');
                return;
            }
            
            const content = `
                <div style="max-height: 400px; overflow-y: auto;">
                    ${modifiedDbs.map(db => `
                        <div style="padding: 10px; border-bottom: 1px solid #e2e8f0;">
                            <div style="display: flex; align-items: center;">
                                <i class="bi bi-database me-2 text-primary"></i>
                                <span style="color: #28a745; font-weight: 500;">${db.database || db.target_database}</span>
                                <i class="bi bi-arrow-right mx-2 text-muted"></i>
                                <span class="text-muted">${db.source_database}</span>
                            </div>
                        </div>
                    `).join('')}
                </div>
            `;
            
            this.showInfoModal('修改的数据库', content);
        },
        
        // 显示修改的表
        showModifiedTables: function() {
            const taskData = window.TaskWizard.taskData;
            const selectedDatabases = taskData.selected_databases || [];
            
            // 收集所有修改的表（按数据库分组）
            const modifiedTablesByDb = {};
            selectedDatabases.forEach(db => {
                const tables = (db.tables || []).filter(t => t.is_modified);
                if (tables.length > 0) {
                    modifiedTablesByDb[db.database || db.target_database] = {
                        sourceDb: db.source_database,
                        targetDb: db.database || db.target_database,
                        tables: tables
                    };
                }
            });
            
            if (Object.keys(modifiedTablesByDb).length === 0) {
                Toast.info('没有修改的表');
                return;
            }
            
            const content = `
                <div style="max-height: 400px; overflow-y: auto;">
                    ${Object.entries(modifiedTablesByDb).map(([dbKey, dbInfo]) => `
                        <div style="margin-bottom: 15px;">
                            <div style="padding: 8px; background: #f8f9fa; border-radius: 6px; margin-bottom: 8px;">
                                <i class="bi bi-database me-2 text-primary"></i>
                                <strong>${dbInfo.targetDb}</strong>
                            </div>
                            ${dbInfo.tables.map(table => `
                                <div style="padding: 8px 8px 8px 30px; border-bottom: 1px solid #f0f0f0;">
                                    <i class="bi bi-table me-2 text-success"></i>
                                    <span style="color: #28a745; font-weight: 500;">${table.target_table}</span>
                                    <i class="bi bi-arrow-right mx-2 text-muted" style="font-size: 12px;"></i>
                                    <span class="text-muted">${table.source_table}</span>
                                </div>
                            `).join('')}
                        </div>
                    `).join('')}
                </div>
            `;
            
            this.showInfoModal('修改的表', content);
        },
        
        // 显示信息模态框
        showInfoModal: function(title, content) {
            const modalHtml = `
                <div id="infoModal" style="display:block; position:fixed; top:0; left:0; width:100%; height:100%; background:rgba(0,0,0,0.6); z-index:2000;">
                    <div style="position:absolute; top:50%; left:50%; transform:translate(-50%,-50%); background:white; padding:0; border-radius:15px; width:600px; max-height:80vh; box-shadow: 0 20px 60px rgba(0,0,0,0.3); display: flex; flex-direction: column;">
                        <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 20px; border-radius: 15px 15px 0 0; flex-shrink: 0;">
                            <h5 style="color: white; margin: 0;"><i class="bi bi-info-circle me-2"></i>${title}</h5>
                        </div>
                        <div style="padding: 20px; overflow-y: auto; flex: 1;">
                            ${content}
                        </div>
                        <div style="padding: 15px 20px; border-top: 1px solid #e2e8f0; text-align: right; flex-shrink: 0;">
                            <button type="button" class="btn btn-primary" onclick="TaskWizardStep4.closeInfoModal()">
                                <i class="bi bi-check-circle"></i> 确定
                            </button>
                        </div>
                    </div>
                </div>
            `;
            
            document.body.insertAdjacentHTML('beforeend', modalHtml);
            
            // 添加ESC键关闭功能
            this.infoModalEscHandler = (e) => {
                if (e.key === 'Escape') {
                    this.closeInfoModal();
                }
            };
            document.addEventListener('keydown', this.infoModalEscHandler);
        },
        
        // 关闭信息模态框
        closeInfoModal: function() {
            const modal = document.getElementById('infoModal');
            if (modal) {
                modal.remove();
            }
            // 移除ESC键监听
            if (this.infoModalEscHandler) {
                document.removeEventListener('keydown', this.infoModalEscHandler);
                this.infoModalEscHandler = null;
            }
        },
        
        // 验证步骤
        validate: async function(taskData) {
            // 最后一步不需要验证
            return true;
        }
    };
})();
