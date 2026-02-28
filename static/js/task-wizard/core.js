// 任务配置向导 - 核心逻辑

// 全局变量
let wizardTaskId = null;
let wizardTask = null;
let wizardStep = 1;
let wizardData = {
    sourceId: '',
    targetId: '',
    selectedDatabases: [],
    syncConfig: {
        batchSize: 2500,
        threadCount: 4,
        syncMode: 'full',
        errorStrategy: 'skip',
        tableExistsStrategy: 'drop'
    }
};
let availableTables = []; // 可选的表列表
let selectedTables = [];  // 已选的表列表
let allDatasources = [];
let hasExistingConfig = false; // 是否有已保存的配置

// 重置向导数据为默认值
function resetWizardData() {
    wizardData = {
        sourceId: '',
        targetId: '',
        selectedDatabases: [],
        syncConfig: {
            batchSize: 2500,
            threadCount: 4,
            syncMode: 'full',
            errorStrategy: 'skip',
            tableExistsStrategy: 'drop'
        }
    };
}

// 显示配置向导
window.showConfigWizard = async function(taskId) {
    wizardTaskId = taskId;
    wizardStep = 1;
    
    // 加载任务信息
    try {
        const response = await fetch(`/api/v1/tasks/${taskId}`);
        const result = await response.json();
        
        if (result.code === 200) {
            wizardTask = result.data;
            document.getElementById('wizardTaskName').textContent = wizardTask.name;
            document.getElementById('wizardSyncDirection').textContent = 
                `${wizardTask.source_type === 'mysql' ? 'MySQL' : 'Elasticsearch'} → ${wizardTask.target_type === 'mysql' ? 'MySQL' : 'Elasticsearch'}`;
            
            // 解析已保存的配置
            if (wizardTask.config && wizardTask.config !== '') {
                try {
                    const savedConfig = JSON.parse(wizardTask.config);
                    
                    hasExistingConfig = true;
                    
                    // 恢复配置数据
                    wizardData.sourceId = savedConfig.source_id || '';
                    wizardData.targetId = savedConfig.target_id || '';
                    wizardData.selectedDatabases = savedConfig.selected_databases || [];
                    wizardData.syncConfig = {
                        batchSize: savedConfig.sync_config?.batch_size || 2500,
                        threadCount: savedConfig.sync_config?.thread_count || 4,
                        syncMode: savedConfig.sync_config?.sync_mode || 'full',
                        errorStrategy: savedConfig.sync_config?.error_strategy || 'skip',
                        tableExistsStrategy: savedConfig.sync_config?.table_exists_strategy || 'drop'
                    };
                } catch (e) {
                    hasExistingConfig = false;
                    resetWizardData();
                }
            } else {
                hasExistingConfig = false;
                resetWizardData();
            }
            
            // 显示模态框
            document.getElementById('configWizardModal').style.display = 'block';
            
            // 加载步骤1
            await loadWizardStep1();
        } else {
            showToast('加载任务失败: ' + result.message, 'error');
        }
    } catch (error) {
        showToast('加载任务失败: ' + error.message, 'error');
    }
};

// 关闭配置向导
window.closeConfigWizard = function() {
    document.getElementById('configWizardModal').style.display = 'none';
    wizardTaskId = null;
    wizardTask = null;
    wizardStep = 1;
    resetWizardData();
    availableTables = [];
    selectedTables = [];
    allDatasources = [];
};

// 显示指定步骤
function showWizardStep(step) {
    wizardStep = step;
    
    // 隐藏所有步骤
    for (let i = 1; i <= 4; i++) {
        document.getElementById(`wizardStep${i}`).style.display = 'none';
        const indicator = document.getElementById(`wizardStepIndicator${i}`);
        indicator.classList.remove('active');
        
        // 如果有已保存的配置，允许跳步骤
        if (hasExistingConfig) {
            indicator.classList.remove('disabled');
        } else {
            // 没有配置，只能访问当前步骤及之前的步骤
            if (i > step) {
                indicator.classList.add('disabled');
            } else {
                indicator.classList.remove('disabled');
                if (i < step) {
                    indicator.classList.add('completed');
                } else {
                    indicator.classList.remove('completed');
                }
            }
        }
    }
    
    // 显示当前步骤
    document.getElementById(`wizardStep${step}`).style.display = 'block';
    document.getElementById(`wizardStepIndicator${step}`).classList.add('active');
    document.getElementById(`wizardStepIndicator${step}`).classList.remove('completed');
    
    // 更新按钮状态
    document.getElementById('wizardPrevBtn').style.display = step === 1 ? 'none' : 'inline-block';
    document.getElementById('wizardNextBtn').style.display = step === 4 ? 'none' : 'inline-block';
    document.getElementById('wizardSaveBtn').style.display = step === 4 ? 'inline-block' : 'none';
}

// 跳转到指定步骤
window.jumpToStep = async function(targetStep) {
    const indicator = document.getElementById(`wizardStepIndicator${targetStep}`);
    
    // 如果步骤被禁用，不允许跳转
    if (indicator.classList.contains('disabled')) {
        showToast('请按顺序完成配置', 'warning');
        return;
    }
    
    // 如果有已保存的配置，允许直接跳转
    if (hasExistingConfig) {
        if (targetStep === 2) {
            await loadWizardStep2();
        } else if (targetStep === 3) {
            loadWizardStep3();
        } else if (targetStep === 4) {
            loadWizardStep4();
        } else {
            showWizardStep(targetStep);
        }
    } else {
        // 没有配置，只能跳到已完成的步骤
        if (targetStep <= wizardStep) {
            showWizardStep(targetStep);
        } else {
            showToast('请按顺序完成配置', 'warning');
        }
    }
};

// 上一步
window.wizardPrevStep = function() {
    if (wizardStep > 1) {
        showWizardStep(wizardStep - 1);
    }
};

// 下一步
window.wizardNextStep = function() {
    if (wizardStep === 1) {
        loadWizardStep2();
    } else if (wizardStep === 2) {
        loadWizardStep3();
    } else if (wizardStep === 3) {
        loadWizardStep4();
    }
};

// 保存配置
window.saveWizardConfig = async function() {
    // 确保配置值正确（转换为后端期望的格式）
    const payload = {
        source_id: wizardData.sourceId,
        target_id: wizardData.targetId,
        selected_databases: wizardData.selectedDatabases,
        sync_config: {
            batch_size: wizardData.syncConfig.batchSize,
            thread_count: wizardData.syncConfig.threadCount,
            sync_mode: wizardData.syncConfig.syncMode,
            error_strategy: wizardData.syncConfig.errorStrategy,
            table_exists_strategy: wizardData.syncConfig.tableExistsStrategy
        }
    };
    
    try {
        const response = await fetch(`/api/v1/tasks/${wizardTaskId}/config`, {
            method: 'PUT',
            headers: {'Content-Type': 'application/json'},
            body: JSON.stringify(payload)
        });
        
        const result = await response.json();
        
        if (result.code === 200) {
            showToast('配置保存成功！', 'success');
            closeConfigWizard();
            loadTaskConfigs();
        } else {
            showToast('保存失败: ' + result.message, 'error');
        }
    } catch (error) {
        showToast('保存失败: ' + error.message, 'error');
    }
};

// 格式化数字（如果 app.js 中没有定义）
if (typeof formatNumber === 'undefined') {
    function formatNumber(num) {
        return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',');
    }
}
