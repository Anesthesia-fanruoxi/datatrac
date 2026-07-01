// 任务配置向导 - 步骤3：同步配置
(function() {
    'use strict';
    
    window.TaskWizardStep3 = {
        // 渲染步骤内容
        render: function(container, taskData) {
            const config = taskData.sync_config || {
                sync_mode: 'full',
                error_strategy: 'skip',
                table_exists_strategy: 'truncate',
                sync_structure_only: false
            };
            
            container.innerHTML = `
                <div>
                    <form id="syncConfigForm">
                        <div class="row">
                            <div class="col-md-6">
                                <div class="mb-4">
                                    <label class="form-label fw-bold mb-3">同步模式 <span class="text-danger">*</span></label>
                                    <div class="d-flex gap-2 flex-wrap">
                                        <div class="form-check-card flex-fill">
                                            <input class="form-check-input" type="radio" name="syncMode" id="syncMode-full" value="full" ${config.sync_mode === 'full' ? 'checked' : ''} required>
                                            <label class="form-check-label d-flex align-items-center" for="syncMode-full">
                                                <i class="bi bi-database-fill-check text-primary me-2"></i>
                                                <div>
                                                    <div>全量同步</div>
                                                    <small class="text-muted">同步所有数据</small>
                                                </div>
                                            </label>
                                        </div>
                                        <div class="form-check-card flex-fill">
                                            <input class="form-check-input" type="radio" name="syncMode" id="syncMode-incremental" value="incremental" ${config.sync_mode === 'incremental' ? 'checked' : ''}>
                                            <label class="form-check-label d-flex align-items-center" for="syncMode-incremental">
                                                <i class="bi bi-arrow-repeat text-success me-2"></i>
                                                <div>
                                                    <div>增量同步</div>
                                                    <small class="text-muted">只同步变更数据</small>
                                                </div>
                                            </label>
                                        </div>
                                        <div class="form-check-card flex-fill">
                                            <input class="form-check-input" type="radio" name="syncMode" id="syncMode-structure" value="structure" ${config.sync_mode === 'structure' ? 'checked' : ''}>
                                            <label class="form-check-label d-flex align-items-center" for="syncMode-structure">
                                                <i class="bi bi-table text-info me-2"></i>
                                                <div>
                                                    <div>只同步表结构</div>
                                                    <small class="text-muted">仅同步表结构，不同步数据</small>
                                                </div>
                                            </label>
                                        </div>
                                    </div>
                                </div>
                                
                                <div class="mb-4" id="errorStrategySection">
                                    <label class="form-label fw-bold mb-3">错误处理策略 <span class="text-danger">*</span></label>
                                    <div class="d-flex gap-2">
                                        <div class="form-check-card flex-fill">
                                            <input class="form-check-input" type="radio" name="errorStrategy" id="errorStrategy-skip" value="skip" ${config.error_strategy === 'skip' ? 'checked' : ''} required>
                                            <label class="form-check-label d-flex align-items-center" for="errorStrategy-skip">
                                                <i class="bi bi-skip-forward text-warning me-2"></i>
                                                <div>
                                                    <div>跳过错误继续</div>
                                                    <small class="text-muted">记录错误并继续</small>
                                                </div>
                                            </label>
                                        </div>
                                        <div class="form-check-card flex-fill">
                                            <input class="form-check-input" type="radio" name="errorStrategy" id="errorStrategy-pause" value="pause" ${config.error_strategy === 'pause' ? 'checked' : ''}>
                                            <label class="form-check-label d-flex align-items-center" for="errorStrategy-pause">
                                                <i class="bi bi-pause-circle text-danger me-2"></i>
                                                <div>
                                                    <div>遇错暂停</div>
                                                    <small class="text-muted">遇到错误立即停止</small>
                                                </div>
                                            </label>
                                        </div>
                                    </div>
                                </div>
                                
                                <div class="mb-4" id="tableStrategySection">
                                    <label class="form-label fw-bold mb-3">目标表存在策略 <span class="text-danger">*</span></label>
                                    <div class="d-flex gap-2">
                                        <div class="form-check-card flex-fill">
                                            <input class="form-check-input" type="radio" name="tableExistsStrategy" id="tableStrategy-drop" value="drop" ${config.table_exists_strategy === 'drop' ? 'checked' : ''} required>
                                            <label class="form-check-label d-flex align-items-center" for="tableStrategy-drop">
                                                <i class="bi bi-trash text-danger me-2"></i>
                                                <div>
                                                    <div>删除重建</div>
                                                    <small class="text-muted">删除旧表并重建</small>
                                                </div>
                                            </label>
                                        </div>
                                        <div class="form-check-card flex-fill">
                                            <input class="form-check-input" type="radio" name="tableExistsStrategy" id="tableStrategy-truncate" value="truncate" ${config.table_exists_strategy === 'truncate' ? 'checked' : ''}>
                                            <label class="form-check-label d-flex align-items-center" for="tableStrategy-truncate">
                                                <i class="bi bi-eraser text-warning me-2"></i>
                                                <div>
                                                    <div>清空数据</div>
                                                    <small class="text-muted">保留表结构清空数据</small>
                                                </div>
                                            </label>
                                        </div>
                                        <div class="form-check-card flex-fill">
                                            <input class="form-check-input" type="radio" name="tableExistsStrategy" id="tableStrategy-backup" value="backup" ${config.table_exists_strategy === 'backup' ? 'checked' : ''}>
                                            <label class="form-check-label d-flex align-items-center" for="tableStrategy-backup">
                                                <i class="bi bi-archive text-info me-2"></i>
                                                <div>
                                                    <div>备份后重建</div>
                                                    <small class="text-muted">备份旧表后重建</small>
                                                </div>
                                            </label>
                                        </div>
                                    </div>
                                </div>
                            </div>
                            
                            <div class="col-md-6">
                                <div class="alert alert-info">
                                    <i class="bi bi-cpu me-2"></i>
                                    <strong>智能自适应配置</strong>
                                    <hr>
                                    <p class="mb-2">系统将根据以下因素自动优化同步参数：</p>
                                    <ul class="mb-0">
                                        <li>CPU核心数和可用内存</li>
                                        <li>表的大小和记录数</li>
                                        <li>字段数量和平均行长度</li>
                                    </ul>
                                </div>
                                
                                <div class="alert alert-success">
                                    <i class="bi bi-lightning-charge me-2"></i>
                                    <strong>性能优化策略</strong>
                                    <hr>
                                    <p class="mb-2"><strong>全量同步：</strong></p>
                                    <ul class="mb-2">
                                        <li>多线程并行处理（自动分配）</li>
                                        <li>批次大小根据表大小自适应</li>
                                    </ul>
                                    <p class="mb-2"><strong>增量同步：</strong></p>
                                    <ul class="mb-2">
                                        <li>Binlog监听单线程（保证顺序）</li>
                                        <li>事件消费自动优化</li>
                                    </ul>
                                    <p class="mb-2"><strong>只同步表结构：</strong></p>
                                    <ul class="mb-0">
                                        <li>对比源表和目标表结构差异</li>
                                        <li>自动执行ALTER语句</li>
                                        <li>新增字段直接ADD</li>
                                    </ul>
                                </div>
                            </div>
                        </div>
                    </form>
                </div>
            `;
            
            // 绑定事件
            this.bindEvents(taskData);
        },
        
        // 绑定事件
        bindEvents: function(taskData) {
            const form = document.getElementById('syncConfigForm');
            
            // 同步模式切换时，隐藏/显示相关选项
            const syncModeRadios = document.querySelectorAll('input[name="syncMode"]');
            const errorStrategySection = document.getElementById('errorStrategySection');
            const tableStrategySection = document.getElementById('tableStrategySection');
            
            const updateVisibility = () => {
                const syncModeRadio = document.querySelector('input[name="syncMode"]:checked');
                const syncMode = syncModeRadio ? syncModeRadio.value : 'full';
                
                if (syncMode === 'structure') {
                    // 只同步表结构模式
                    if (errorStrategySection) errorStrategySection.style.display = 'none';
                    if (tableStrategySection) tableStrategySection.style.display = 'none';
                } else {
                    // 全量/增量同步模式
                    if (errorStrategySection) errorStrategySection.style.display = 'block';
                    if (tableStrategySection) tableStrategySection.style.display = 'block';
                }
            };
            
            syncModeRadios.forEach(radio => {
                radio.addEventListener('change', updateVisibility);
            });
            
            // 初始化可见性
            updateVisibility();
            
            // 实时保存配置
            form.addEventListener('change', function() {
                const syncModeRadio = document.querySelector('input[name="syncMode"]:checked');
                const errorStrategyRadio = document.querySelector('input[name="errorStrategy"]:checked');
                const tableStrategyRadio = document.querySelector('input[name="tableExistsStrategy"]:checked');
                
                const syncMode = syncModeRadio ? syncModeRadio.value : 'full';
                
                taskData.sync_config = {
                    sync_mode: syncMode,
                    sync_structure_only: syncMode === 'structure',
                    error_strategy: errorStrategyRadio ? errorStrategyRadio.value : 'skip',
                    table_exists_strategy: tableStrategyRadio ? tableStrategyRadio.value : 'truncate'
                };
            });
        },
        
        // 验证步骤
        validate: async function(taskData) {
            const form = document.getElementById('syncConfigForm');
            
            if (!form.checkValidity()) {
                form.reportValidity();
                return false;
            }
            
            const syncModeRadio = document.querySelector('input[name="syncMode"]:checked');
            const errorStrategyRadio = document.querySelector('input[name="errorStrategy"]:checked');
            const tableStrategyRadio = document.querySelector('input[name="tableExistsStrategy"]:checked');
            
            const syncMode = syncModeRadio ? syncModeRadio.value : 'full';
            
            // 保存配置
            taskData.sync_config = {
                sync_mode: syncMode,
                sync_structure_only: syncMode === 'structure',
                error_strategy: errorStrategyRadio ? errorStrategyRadio.value : 'skip',
                table_exists_strategy: tableStrategyRadio ? tableStrategyRadio.value : 'truncate'
            };
            
            return true;
        }
    };
})();
