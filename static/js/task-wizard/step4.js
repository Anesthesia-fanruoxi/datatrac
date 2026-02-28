// 任务配置向导 - 步骤4：确认配置

// 加载步骤4：确认配置
function loadWizardStep4() {
    // 先确保步骤3的表单已初始化（如果用户直接跳到步骤4）
    const batchSizeInput = document.getElementById('wizardBatchSize');
    const threadCountInput = document.getElementById('wizardThreadCount');
    
    // 如果表单元素存在但值为空，说明表单未初始化，需要填充默认值
    if (batchSizeInput && (!batchSizeInput.value || batchSizeInput.value.trim() === '')) {
        batchSizeInput.value = wizardData.syncConfig.batchSize;
    }
    if (threadCountInput && (!threadCountInput.value || threadCountInput.value.trim() === '')) {
        threadCountInput.value = wizardData.syncConfig.threadCount;
    }
    
    // 检查单选按钮是否有选中项，如果没有则设置默认选中
    const syncModeRadio = document.querySelector('input[name="wizardSyncMode"]:checked');
    if (!syncModeRadio) {
        const defaultSyncModeRadio = document.querySelector(`input[name="wizardSyncMode"][value="${wizardData.syncConfig.syncMode}"]`);
        if (defaultSyncModeRadio) {
            defaultSyncModeRadio.checked = true;
        }
    }
    
    const errorStrategyRadio = document.querySelector('input[name="wizardErrorStrategy"]:checked');
    if (!errorStrategyRadio) {
        const defaultErrorRadio = document.querySelector(`input[name="wizardErrorStrategy"][value="${wizardData.syncConfig.errorStrategy}"]`);
        if (defaultErrorRadio) {
            defaultErrorRadio.checked = true;
        }
    }
    
    const tableStrategyRadio = document.querySelector('input[name="wizardTableExistsStrategy"]:checked');
    if (!tableStrategyRadio) {
        const defaultTableRadio = document.querySelector(`input[name="wizardTableExistsStrategy"][value="${wizardData.syncConfig.tableExistsStrategy}"]`);
        if (defaultTableRadio) {
            defaultTableRadio.checked = true;
        }
    }
    
    // 现在收集配置 - 从表单读取
    if (batchSizeInput && batchSizeInput.value && batchSizeInput.value.trim() !== '') {
        const parsedBatchSize = parseInt(batchSizeInput.value);
        if (!isNaN(parsedBatchSize) && parsedBatchSize > 0) {
            wizardData.syncConfig.batchSize = parsedBatchSize;
        }
    }
    
    if (threadCountInput && threadCountInput.value && threadCountInput.value.trim() !== '') {
        const parsedThreadCount = parseInt(threadCountInput.value);
        if (!isNaN(parsedThreadCount) && parsedThreadCount > 0) {
            wizardData.syncConfig.threadCount = parsedThreadCount;
        }
    }
    
    // 获取单选按钮的选中值
    const finalSyncModeRadio = document.querySelector('input[name="wizardSyncMode"]:checked');
    if (finalSyncModeRadio && finalSyncModeRadio.value) {
        wizardData.syncConfig.syncMode = finalSyncModeRadio.value;
    }
    
    const finalErrorStrategyRadio = document.querySelector('input[name="wizardErrorStrategy"]:checked');
    if (finalErrorStrategyRadio && finalErrorStrategyRadio.value) {
        wizardData.syncConfig.errorStrategy = finalErrorStrategyRadio.value;
    }
    
    const finalTableStrategyRadio = document.querySelector('input[name="wizardTableExistsStrategy"]:checked');
    if (finalTableStrategyRadio && finalTableStrategyRadio.value) {
        wizardData.syncConfig.tableExistsStrategy = finalTableStrategyRadio.value;
    }
    
    // 显示摘要
    const totalTables = wizardData.selectedDatabases.reduce((sum, db) => sum + db.tables.length, 0);
    const modifiedTables = wizardData.selectedDatabases.reduce((sum, db) => {
        return sum + db.tables.filter(t => t.is_modified || t.source_table !== t.target_table).length;
    }, 0);
    
    document.getElementById('summarySourceId').textContent = document.querySelector(`#wizardSourceId option[value="${wizardData.sourceId}"]`).textContent;
    document.getElementById('summaryTargetId').textContent = document.querySelector(`#wizardTargetId option[value="${wizardData.targetId}"]`).textContent;
    
    // 显示表数量和修改统计
    if (modifiedTables > 0) {
        document.getElementById('summaryTableCount').innerHTML = `${totalTables} 张表 <span style="color: #3182ce;">(${modifiedTables} 个已修改)</span>`;
    } else {
        document.getElementById('summaryTableCount').textContent = `${totalTables} 张表`;
    }
    
    document.getElementById('summaryBatchSize').textContent = wizardData.syncConfig.batchSize;
    document.getElementById('summaryThreadCount').textContent = wizardData.syncConfig.threadCount;
    
    // 显示同步模式
    const syncModeText = {
        'full': '全量同步',
        'incremental': '增量同步'
    };
    document.getElementById('summarySyncMode').textContent = syncModeText[wizardData.syncConfig.syncMode] || wizardData.syncConfig.syncMode;
    
    // 显示策略信息
    const errorStrategyText = {
        'skip': '跳过错误继续',
        'pause': '遇错暂停'
    };
    document.getElementById('summaryErrorStrategy').textContent = errorStrategyText[wizardData.syncConfig.errorStrategy] || wizardData.syncConfig.errorStrategy;
    
    const tableStrategyText = {
        'drop': '删除重建',
        'truncate': '清空数据',
        'backup': '备份后重建'
    };
    document.getElementById('summaryTableExistsStrategy').textContent = tableStrategyText[wizardData.syncConfig.tableExistsStrategy] || wizardData.syncConfig.tableExistsStrategy;
    
    // 显示选中的表（树形结构）
    const tableListHtml = wizardData.selectedDatabases.map(db => {
        // 检查数据库名是否被修改
        const isDatabaseModified = db.is_database_modified || (db.source_database && db.source_database !== db.database);
        const databaseDisplayName = isDatabaseModified 
            ? `<span style="color: #3182ce;">${db.database}</span><span style="color: #718096; font-size: 12px;">(原名：${db.source_database || db.database})</span>`
            : db.database;
        
        const tablesHtml = db.tables.map(tableConfig => {
            const isModified = tableConfig.is_modified || tableConfig.source_table !== tableConfig.target_table;
            if (isModified) {
                return `
                <div style="padding: 4px 0 4px 40px; font-size: 13px;">
                    <i class="bi bi-table me-1" style="color: #764ba2; font-size: 12px;"></i>
                    <span style="color: #3182ce;">${tableConfig.target_table}</span><span style="color: #718096; font-size: 12px;">(原名：${tableConfig.source_table})</span>
                </div>`;
            } else {
                return `
                <div style="padding: 4px 0 4px 40px; font-size: 13px; color: #4a5568;">
                    <i class="bi bi-table me-1" style="color: #764ba2; font-size: 12px;"></i>${tableConfig.source_table}
                </div>`;
            }
        }).join('');
        
        return `
        <div style="border-bottom: 1px solid #e9ecef;">
            <div style="padding: 10px; background: #f8f9fa; cursor: pointer; display: flex; align-items: center;" onclick="toggleSummaryDatabase('${db.database}')">
                <i class="bi bi-chevron-right me-2" id="summary-chevron-${db.database}" style="transition: transform 0.3s; color: #667eea; font-size: 12px;"></i>
                <i class="bi bi-database me-2" style="color: #667eea; font-size: 14px;"></i>
                <span style="font-weight: 600; font-size: 14px;">${databaseDisplayName}</span>
                <span style="color: #718096; font-size: 12px; margin-left: 8px;">(${db.tables.length})</span>
            </div>
            <div id="summary-tables-${db.database}" style="display: none;">
                ${tablesHtml}
            </div>
        </div>
        `;
    }).join('');
    document.getElementById('summaryTableList').innerHTML = tableListHtml;
    
    showWizardStep(4);
}


// 切换摘要页面数据库展开/折叠
window.toggleSummaryDatabase = function(database) {
    const tablesDiv = document.getElementById(`summary-tables-${database}`);
    const chevron = document.getElementById(`summary-chevron-${database}`);
    
    if (tablesDiv && chevron) {
        if (tablesDiv.style.display === 'none') {
            tablesDiv.style.display = 'block';
            chevron.style.transform = 'rotate(90deg)';
        } else {
            tablesDiv.style.display = 'none';
            chevron.style.transform = 'rotate(0deg)';
        }
    }
};
