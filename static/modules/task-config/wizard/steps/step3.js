// 任务配置向导 - 步骤3：同步配置
(function() {
    'use strict';
    
    window.TaskWizardStep3 = {
        // 渲染步骤内容
        render: function(container, taskData) {
            const config = taskData.sync_config || {
                sync_mode: 'full',
                batch_size: 1000,
                thread_count: 4,
                error_strategy: 'skip',
                table_exists_strategy: 'truncate'
            };
            
            container.innerHTML = `
                <div>
                    <form id="syncConfigForm">
                        <div class="row">
                            <div class="col-md-6">
                                <div class="mb-4">
                                    <label class="form-label fw-bold mb-3">同步模式 <span class="text-danger">*</span></label>
                                    <div class="d-flex gap-2">
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
                                    </div>
                                </div>
                                
                                <div class="mb-4">
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
                                
                                <div class="mb-4">
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
                                <div class="mb-4">
                                    <label class="form-label fw-bold">批次大小 <span class="text-danger">*</span></label>
                                    <input type="number" class="form-control" id="batchSize" 
                                        value="${config.batch_size}" min="100" max="10000" required>
                                    <div class="form-text">每批次同步的记录数，建议 1000-5000</div>
                                </div>
                                
                                <div class="mb-4">
                                    <label class="form-label fw-bold">并发线程数 <span class="text-danger">*</span></label>
                                    <input type="number" class="form-control" id="threadCount" 
                                        value="${config.thread_count}" min="1" max="16" required>
                                    <div class="form-text">并发同步的线程数，建议 2-8</div>
                                </div>
                                
                                <div class="alert alert-info">
                                    <i class="bi bi-info-circle me-2"></i>
                                    <strong>提示：</strong>配置参数会影响同步性能和数据安全，请根据实际情况调整
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
            
            // 实时保存配置
            form.addEventListener('change', function() {
                const syncModeRadio = document.querySelector('input[name="syncMode"]:checked');
                const errorStrategyRadio = document.querySelector('input[name="errorStrategy"]:checked');
                const tableStrategyRadio = document.querySelector('input[name="tableExistsStrategy"]:checked');
                
                taskData.sync_config = {
                    sync_mode: syncModeRadio ? syncModeRadio.value : 'full',
                    batch_size: parseInt(document.getElementById('batchSize').value),
                    thread_count: parseInt(document.getElementById('threadCount').value),
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
            
            // 保存配置
            taskData.sync_config = {
                sync_mode: syncModeRadio ? syncModeRadio.value : 'full',
                batch_size: parseInt(document.getElementById('batchSize').value),
                thread_count: parseInt(document.getElementById('threadCount').value),
                error_strategy: errorStrategyRadio ? errorStrategyRadio.value : 'skip',
                table_exists_strategy: tableStrategyRadio ? tableStrategyRadio.value : 'truncate'
            };
            
            return true;
        }
    };
})();
