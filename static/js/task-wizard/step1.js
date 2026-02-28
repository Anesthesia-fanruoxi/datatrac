// 任务配置向导 - 步骤1：选择数据源

// 加载步骤1：选择数据源
async function loadWizardStep1() {
    try {
        const response = await fetch('/api/v1/datasources');
        const result = await response.json();
        
        if (result.code === 200) {
            allDatasources = result.data || [];
            
            // 渲染数据源下拉框
            renderDatasourceSelects();
            
            // 监听源数据源变化
            document.getElementById('wizardSourceId').addEventListener('change', function() {
                renderDatasourceSelects();
            });
            
            // 监听目标数据源变化
            document.getElementById('wizardTargetId').addEventListener('change', function() {
                renderDatasourceSelects();
            });
        }
    } catch (error) {
        showToast('加载数据源失败: ' + error.message, 'error');
    }
    
    showWizardStep(1);
}

// 渲染数据源下拉框（互斥选择）
function renderDatasourceSelects() {
    const sourceSelect = document.getElementById('wizardSourceId');
    const targetSelect = document.getElementById('wizardTargetId');
    
    const currentSourceId = wizardData.sourceId || sourceSelect.value;
    const currentTargetId = wizardData.targetId || targetSelect.value;
    
    // 过滤源数据源（排除已选的目标数据源）
    const sourceDatasources = allDatasources.filter(ds => 
        ds.type === wizardTask.source_type && ds.id !== currentTargetId
    );
    sourceSelect.innerHTML = '<option value="">请选择源数据源</option>' +
        sourceDatasources.map(ds => `<option value="${ds.id}" ${ds.id === currentSourceId ? 'selected' : ''}>${ds.name} (${ds.host}:${ds.port})</option>`).join('');
    
    // 过滤目标数据源（排除已选的源数据源）
    const targetDatasources = allDatasources.filter(ds => 
        ds.type === wizardTask.target_type && ds.id !== currentSourceId
    );
    targetSelect.innerHTML = '<option value="">请选择目标数据源</option>' +
        targetDatasources.map(ds => `<option value="${ds.id}" ${ds.id === currentTargetId ? 'selected' : ''}>${ds.name} (${ds.host}:${ds.port})</option>`).join('');
    
    // 设置选中值（确保回显）
    if (currentSourceId) {
        sourceSelect.value = currentSourceId;
    }
    if (currentTargetId) {
        targetSelect.value = currentTargetId;
    }
}
