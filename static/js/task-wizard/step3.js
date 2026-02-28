// 任务配置向导 - 步骤3：配置参数

// 加载步骤3：配置参数
function loadWizardStep3() {
    // 检查是否选择了表（优先检查 wizardData.selectedDatabases，如果为空再检查 selectedTables）
    if (wizardData.selectedDatabases.length === 0 && selectedTables.length === 0) {
        showToast('请至少选择一张表', 'warning');
        return;
    }
    
    // 如果 selectedTables 有数据，则需要转换为 wizardData.selectedDatabases 格式
    if (selectedTables.length > 0) {
        // 按数据库分组，并保存表名映射关系
        const databaseMap = {};
        selectedTables.forEach(item => {
            if (!databaseMap[item.database]) {
                databaseMap[item.database] = {
                    source_database: item.sourceDatabase || item.database,
                    is_database_modified: item.isDatabaseModified || false,
                    tables: []
                };
            }
            databaseMap[item.database].tables.push({
                source_table: item.table,
                target_table: item.targetTable || item.table,
                is_modified: item.isModified || false
            });
        });
        
        // 转换为数组
        wizardData.selectedDatabases = Object.keys(databaseMap).map(db => ({
            database: db,
            source_database: databaseMap[db].source_database,
            is_database_modified: databaseMap[db].is_database_modified,
            tables: databaseMap[db].tables
        }));
    }
    // 如果 wizardData.selectedDatabases 已有数据（从已保存的配置加载），则直接使用
    
    // 填充配置表单
    document.getElementById('wizardBatchSize').value = wizardData.syncConfig.batchSize;
    document.getElementById('wizardThreadCount').value = wizardData.syncConfig.threadCount;
    
    // 设置同步模式单选按钮的选中状态
    const syncModeRadios = document.getElementsByName('wizardSyncMode');
    syncModeRadios.forEach(radio => {
        radio.checked = (radio.value === wizardData.syncConfig.syncMode);
    });
    
    // 设置单选按钮的选中状态
    const errorStrategyRadios = document.getElementsByName('wizardErrorStrategy');
    errorStrategyRadios.forEach(radio => {
        radio.checked = (radio.value === wizardData.syncConfig.errorStrategy);
    });
    
    const tableStrategyRadios = document.getElementsByName('wizardTableExistsStrategy');
    tableStrategyRadios.forEach(radio => {
        radio.checked = (radio.value === wizardData.syncConfig.tableExistsStrategy);
    });
    
    showWizardStep(3);
}
