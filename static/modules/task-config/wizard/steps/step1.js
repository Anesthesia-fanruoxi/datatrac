// 任务配置向导 - 步骤1：选择数据源
(function() {
    'use strict';
    
    window.TaskWizardStep1 = {
        sourceDatasources: [],
        targetDatasources: [],
        
        // 渲染步骤内容
        render: async function(container, taskData) {
            // 检查任务数据是否已加载
            if (!taskData.task) {
                container.innerHTML = '<div class="alert alert-warning">任务信息加载中，请稍候...</div>';
                return;
            }
            
            container.innerHTML = '<div class="text-center"><div class="spinner-border"></div><div>加载数据源列表...</div></div>';
            
            try {
                // 加载数据源列表
                const result = await HttpUtils.get('/api/v1/datasources');
                
                if (result.code !== 200) {
                    container.innerHTML = '<div class="alert alert-danger">加载数据源失败: ' + result.message + '</div>';
                    return;
                }
                
                const datasources = result.data || [];
                const task = taskData.task;
                
                // 过滤并保存数据源
                this.sourceDatasources = datasources.filter(ds => ds.type === task.source_type);
                this.targetDatasources = datasources.filter(ds => ds.type === task.target_type);
                
                container.innerHTML = `
                    <div class="row">
                        <div class="col-md-6">
                            <div class="mb-4">
                                <label class="form-label">
                                    <i class="bi bi-database me-2"></i>选择源数据源
                                    <span class="text-danger">*</span>
                                </label>
                                <p class="text-muted small mb-2">类型: ${task.source_type === 'mysql' ? 'MySQL' : 'Elasticsearch'}</p>
                                <div id="source-select-container">
                                    ${this.renderDatasourceSelect(this.sourceDatasources, 'source', taskData.source_id, taskData.target_id)}
                                </div>
                            </div>
                        </div>
                        <div class="col-md-6">
                            <div class="mb-4">
                                <label class="form-label">
                                    <i class="bi bi-database me-2"></i>选择目标数据源
                                    <span class="text-danger">*</span>
                                </label>
                                <p class="text-muted small mb-2">类型: ${task.target_type === 'mysql' ? 'MySQL' : 'Elasticsearch'}</p>
                                <div id="target-select-container">
                                    ${this.renderDatasourceSelect(this.targetDatasources, 'target', taskData.target_id, taskData.source_id)}
                                </div>
                            </div>
                        </div>
                    </div>
                    
                    <div class="alert alert-info mt-4">
                        <i class="bi bi-info-circle me-2"></i>
                        请选择源数据源和目标数据源，然后点击"下一步"继续配置
                    </div>
                `;
                
                // 绑定事件
                this.bindEvents(taskData);
                
            } catch (error) {
                container.innerHTML = '<div class="alert alert-danger">加载失败: ' + error.message + '</div>';
            }
        },
        
        // 渲染数据源下拉框
        renderDatasourceSelect: function(datasources, type, selectedId, excludeId) {
            if (datasources.length === 0) {
                return '<div class="alert alert-warning">暂无可用数据源，请先到"数据源配置"页面创建数据源</div>';
            }
            
            // 过滤掉已被另一方选择的数据源
            const filteredDatasources = excludeId 
                ? datasources.filter(ds => ds.id !== excludeId)
                : datasources;
            
            if (filteredDatasources.length === 0) {
                return '<div class="alert alert-warning">没有可用的数据源（已被另一方选择）</div>';
            }
            
            const options = filteredDatasources.map(ds => {
                const selected = selectedId === ds.id ? 'selected' : '';
                return `<option value="${ds.id}" ${selected}>${ds.name} (${ds.host}:${ds.port})</option>`;
            }).join('');
            
            return `
                <select class="form-select form-select-lg" id="datasource-${type}" data-type="${type}">
                    <option value="">请选择数据源</option>
                    ${options}
                </select>
            `;
        },
        
        // 更新目标数据源下拉框
        updateTargetSelect: function(taskData) {
            const container = document.getElementById('target-select-container');
            if (container) {
                container.innerHTML = this.renderDatasourceSelect(
                    this.targetDatasources, 
                    'target', 
                    taskData.target_id, 
                    taskData.source_id
                );
                
                // 重新绑定目标下拉框事件
                const targetSelect = document.getElementById('datasource-target');
                if (targetSelect) {
                    targetSelect.addEventListener('change', () => {
                        taskData.target_id = targetSelect.value;
                        this.updateSourceSelect(taskData);
                    });
                }
            }
        },
        
        // 更新源数据源下拉框
        updateSourceSelect: function(taskData) {
            const container = document.getElementById('source-select-container');
            if (container) {
                container.innerHTML = this.renderDatasourceSelect(
                    this.sourceDatasources, 
                    'source', 
                    taskData.source_id, 
                    taskData.target_id
                );
                
                // 重新绑定源下拉框事件
                const sourceSelect = document.getElementById('datasource-source');
                if (sourceSelect) {
                    sourceSelect.addEventListener('change', () => {
                        taskData.source_id = sourceSelect.value;
                        this.updateTargetSelect(taskData);
                    });
                }
            }
        },
        
        // 绑定事件
        bindEvents: function(taskData) {
            // 监听源数据源选择
            const sourceSelect = document.getElementById('datasource-source');
            if (sourceSelect) {
                sourceSelect.addEventListener('change', () => {
                    taskData.source_id = sourceSelect.value;
                    // 更新目标数据源下拉框，排除已选择的源
                    this.updateTargetSelect(taskData);
                });
            }
            
            // 监听目标数据源选择
            const targetSelect = document.getElementById('datasource-target');
            if (targetSelect) {
                targetSelect.addEventListener('change', () => {
                    taskData.target_id = targetSelect.value;
                    // 更新源数据源下拉框，排除已选择的目标
                    this.updateSourceSelect(taskData);
                });
            }
        },
        
        // 验证步骤
        validate: async function(taskData) {
            if (!taskData.source_id) {
                Toast.warning('请选择源数据源');
                return false;
            }
            
            if (!taskData.target_id) {
                Toast.warning('请选择目标数据源');
                return false;
            }
            
            if (taskData.source_id === taskData.target_id) {
                Toast.warning('源数据源和目标数据源不能相同');
                return false;
            }
            
            return true;
        }
    };
})();
