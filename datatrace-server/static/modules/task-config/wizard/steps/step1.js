// 任务配置向导 - 步骤1：选择数据源
(function() {
    'use strict';
    
    function escapeHtml(str) {
        if (str == null) return '';
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }
    
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
                                    ${this.renderSourceDatasourceSelect(this.sourceDatasources, taskData.source_id)}
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
                                <p class="text-muted small mb-2">点击卡片多选目标数据源</p>
                                <div id="target-select-container">
                                    ${this.renderTargetDatasourceSelect(this.targetDatasources, taskData.target_ids, taskData.source_id)}
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
        
        // 渲染源数据源下拉框（单选）
        renderSourceDatasourceSelect: function(datasources, selectedId) {
            if (datasources.length === 0) {
                return '<div class="alert alert-warning">暂无可用数据源，请先到"数据源配置"页面创建数据源</div>';
            }
            
            const options = datasources.map(ds => {
                const selected = selectedId === ds.id ? 'selected' : '';
                return `<option value="${ds.id}" ${selected}>${ds.name} (${ds.host}:${ds.port})</option>`;
            }).join('');
            
            return `
                <select class="form-select form-select-lg" id="datasource-source" data-type="source">
                    <option value="">请选择数据源</option>
                    ${options}
                </select>
            `;
        },
        
        // 渲染目标数据源卡片（多选）
        renderTargetDatasourceSelect: function(datasources, selectedIds, excludeId) {
            if (datasources.length === 0) {
                return '<div class="alert alert-warning">暂无可用数据源，请先到"数据源配置"页面创建数据源</div>';
            }
            
            // 过滤掉源数据源
            const availableDatasources = excludeId
                ? datasources.filter(ds => ds.id !== excludeId)
                : datasources;
            
            if (availableDatasources.length === 0) {
                return '<div class="alert alert-warning">没有可用的目标数据源</div>';
            }
            
            // 处理selectedIds（兼容旧的target_id）
            if (!selectedIds || selectedIds.length === 0) {
                selectedIds = [];
            }
            
            const selectedIdSet = new Set((selectedIds || []).map(String));
            const cardsHtml = availableDatasources.map(ds => {
                const selected = selectedIdSet.has(String(ds.id));
                const cardClass = selected
                    ? 'target-ds-card border-primary bg-primary bg-opacity-10 shadow-sm'
                    : 'target-ds-card border';
                const checkIcon = selected ? '<i class="bi bi-check-circle-fill text-primary position-absolute top-0 end-0 m-2"></i>' : '';
                return `
                    <div class="target-ds-card-wrapper col-6 col-lg-12 mb-2" data-ds-id="${ds.id}">
                        <div class="card ${cardClass} target-ds-card-item h-100 position-relative target-ds-card-clickable"
                             style="transition: all 0.2s ease; min-height: 64px; cursor: pointer;">
                            ${checkIcon}
                            <div class="card-body py-2 px-3">
                                <div class="d-flex align-items-center">
                                    <i class="bi bi-database text-secondary me-2" style="font-size: 1.25rem;"></i>
                                    <div class="flex-grow-1 min-w-0">
                                        <div class="fw-medium text-truncate">${escapeHtml(ds.name)}</div>
                                        <small class="text-muted">${escapeHtml(ds.host)}:${ds.port}</small>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                `;
            }).join('');
            
            return `
                <div class="row g-2" id="datasource-target-cards">
                    ${cardsHtml}
                </div>
                <small class="text-muted d-block mt-2">已选 <span id="target-ds-selected-count">${selectedIds.length}</span> 个</small>
            `;
        },
        
        // 绑定事件
        bindEvents: function(taskData) {
            // 初始化target_ids
            if (!taskData.target_ids) {
                taskData.target_ids = [];
            }
            
            // 监听源数据源选择
            const sourceSelect = document.getElementById('datasource-source');
            if (sourceSelect) {
                sourceSelect.addEventListener('change', () => {
                    taskData.source_id = sourceSelect.value;
                    // 更新目标数据源下拉框，排除已选择的源
                    this.updateTargetSelect(taskData);
                });
            }
            
            // 监听目标数据源卡片点击（多选）
            this.bindTargetCardEvents(taskData);
        },
        
        // 绑定目标数据源卡片点击事件
        bindTargetCardEvents: function(taskData) {
            document.querySelectorAll('.target-ds-card-item').forEach(card => {
                card.addEventListener('click', () => {
                    const wrapper = card.closest('.target-ds-card-wrapper');
                    const dsId = wrapper.getAttribute('data-ds-id');
                    const idx = (taskData.target_ids || []).indexOf(dsId);
                    if (idx >= 0) {
                        taskData.target_ids.splice(idx, 1);
                    } else {
                        if (!taskData.target_ids) taskData.target_ids = [];
                        taskData.target_ids.push(dsId);
                    }
                    taskData.target_id = (taskData.target_ids && taskData.target_ids[0]) || '';
                    this.updateTargetCardUi(taskData);
                });
            });
        },
        
        // 更新目标数据源卡片选中状态与计数
        updateTargetCardUi: function(taskData) {
            const selectedIds = taskData.target_ids || [];
            const selectedSet = new Set(selectedIds.map(String));
            document.querySelectorAll('.target-ds-card-wrapper').forEach(wrapper => {
                const dsId = wrapper.getAttribute('data-ds-id');
                const card = wrapper.querySelector('.target-ds-card-item');
                const selected = selectedSet.has(dsId);
                if (selected) {
                    card.classList.add('border-primary', 'bg-primary', 'bg-opacity-10', 'shadow-sm');
                    card.classList.remove('border');
                    if (!card.querySelector('.bi-check-circle-fill')) {
                        const icon = document.createElement('i');
                        icon.className = 'bi bi-check-circle-fill text-primary position-absolute top-0 end-0 m-2';
                        card.appendChild(icon);
                    }
                } else {
                    card.classList.remove('border-primary', 'bg-primary', 'bg-opacity-10', 'shadow-sm');
                    card.classList.add('border');
                    const icon = card.querySelector('.bi-check-circle-fill');
                    if (icon) icon.remove();
                }
            });
            const countEl = document.getElementById('target-ds-selected-count');
            if (countEl) countEl.textContent = selectedIds.length;
        },
        
        // 更新目标数据源卡片区域
        updateTargetSelect: function(taskData) {
            const container = document.getElementById('target-select-container');
            if (container) {
                container.innerHTML = this.renderTargetDatasourceSelect(
                    this.targetDatasources,
                    taskData.target_ids,
                    taskData.source_id
                );
                this.bindTargetCardEvents(taskData);
            }
        },
        
        // 验证步骤
        validate: async function(taskData) {
            if (!taskData.source_id) {
                Toast.warning('请选择源数据源');
                return false;
            }
            
            if (!taskData.target_ids || taskData.target_ids.length === 0) {
                Toast.warning('请至少选择一个目标数据源');
                return false;
            }
            
            if (taskData.target_ids.includes(taskData.source_id)) {
                Toast.warning('源数据源和目标数据源不能相同');
                return false;
            }
            
            return true;
        }
    };
})();
