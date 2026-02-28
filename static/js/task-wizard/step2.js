// 任务配置向导 - 步骤2：选择库和表

// 加载步骤2：选择库和表（穿梭框）
async function loadWizardStep2() {
    wizardData.sourceId = document.getElementById('wizardSourceId').value;
    wizardData.targetId = document.getElementById('wizardTargetId').value;
    
    if (!wizardData.sourceId || !wizardData.targetId) {
        showToast('请选择源和目标数据源', 'warning');
        return;
    }
    
    if (wizardData.sourceId === wizardData.targetId) {
        showToast('源和目标不能是同一个数据源', 'warning');
        return;
    }
    
    // 先显示步骤2，显示加载中
    showWizardStep(2);
    
    // 显示加载中状态
    document.getElementById('availableList').innerHTML = '<div class="text-center text-muted p-4"><div class="spinner-border spinner-border-sm me-2" role="status"></div>正在加载数据库和表...</div>';
    
    // 加载所有数据库和表
    try {
        const dbResponse = await fetch(`/api/v1/datasources/${wizardData.sourceId}/databases`);
        const dbResult = await dbResponse.json();
        
        if (dbResult.code !== 200) {
            showToast('加载数据库失败: ' + dbResult.message, 'error');
            document.getElementById('availableList').innerHTML = '<div class="text-center text-danger p-4"><i class="bi bi-exclamation-triangle"></i><div class="mt-2">加载失败</div></div>';
            return;
        }
        
        const databases = dbResult.data || [];
        
        if (databases.length === 0) {
            showToast('该数据源没有数据库', 'warning');
            availableTables = [];
            selectedTables = [];
            renderTransferLists();
            return;
        }
        
        availableTables = [];
        
        // 加载所有数据库的表
        for (const db of databases) {
            try {
                const tableResponse = await fetch(`/api/v1/datasources/${wizardData.sourceId}/tables?database=${encodeURIComponent(db.name)}`);
                const tableResult = await tableResponse.json();
                
                if (tableResult.code === 200) {
                    const tables = tableResult.data || [];
                    
                    if (tables.length > 0) {
                        tables.forEach(table => {
                            availableTables.push({
                                database: db.name,
                                sourceDatabase: db.name,  // 保存原始数据库名
                                table: table.name,
                                targetTable: table.name,  // 默认目标表名与源表名相同
                                isModified: false,
                                isDatabaseModified: false,
                                key: `${db.name}.${table.name}`
                            });
                        });
                    }
                }
            } catch (error) {
                // 忽略单个数据库加载失败
            }
        }
        
        if (availableTables.length === 0) {
            showToast('所有数据库都没有表', 'warning');
        }
        
        // 恢复已选的表（从保存的配置中）
        selectedTables = [];
        if (wizardData.selectedDatabases && wizardData.selectedDatabases.length > 0) {
            wizardData.selectedDatabases.forEach(dbSelection => {
                if (dbSelection.tables) {
                    // 使用源数据库名来匹配（因为 availableTables 使用的是源数据库名）
                    const sourceDbName = dbSelection.source_database || dbSelection.database;
                    
                    // 兼容旧格式（数组）和新格式（对象数组）
                    dbSelection.tables.forEach(tableConfig => {
                        let sourceTable, targetTable, isModified;
                        
                        if (typeof tableConfig === 'string') {
                            // 旧格式：直接是表名字符串
                            sourceTable = tableConfig;
                            targetTable = tableConfig;
                            isModified = false;
                        } else {
                            // 新格式：包含映射关系的对象
                            sourceTable = tableConfig.source_table || tableConfig.table;
                            targetTable = tableConfig.target_table || sourceTable;
                            isModified = tableConfig.is_modified || false;
                        }
                        
                        // 使用源数据库名来查找表
                        const tableItem = availableTables.find(t => 
                            t.database === sourceDbName && t.table === sourceTable
                        );
                        if (tableItem) {
                            selectedTables.push({
                                ...tableItem,
                                database: dbSelection.database,  // 使用目标数据库名
                                sourceDatabase: sourceDbName,    // 保存源数据库名
                                targetTable: targetTable,
                                isModified: isModified,
                                isDatabaseModified: dbSelection.is_database_modified || false
                            });
                        }
                    });
                }
            });
        }
        
        // 渲染穿梭框
        renderTransferLists();
        
        // 绑定搜索事件
        const searchAvailableInput = document.getElementById('searchAvailable');
        const searchSelectedInput = document.getElementById('searchSelected');
        
        if (searchAvailableInput && searchSelectedInput) {
            searchAvailableInput.removeEventListener('input', handleSearchAvailable);
            searchSelectedInput.removeEventListener('input', handleSearchSelected);
            
            searchAvailableInput.addEventListener('input', handleSearchAvailable);
            searchSelectedInput.addEventListener('input', handleSearchSelected);
        }
        
    } catch (error) {
        showToast('加载数据失败: ' + error.message, 'error');
        document.getElementById('availableList').innerHTML = '<div class="text-center text-danger p-4"><i class="bi bi-exclamation-triangle"></i><div class="mt-2">加载失败: ' + error.message + '</div></div>';
        return;
    }
}

// 搜索处理函数
function handleSearchAvailable() {
    renderTransferLists();
}

function handleSearchSelected() {
    renderTransferLists();
}

// 渲染穿梭框列表
function renderTransferLists() {
    const searchAvailable = (document.getElementById('searchAvailable')?.value || '').toLowerCase();
    const searchSelected = (document.getElementById('searchSelected')?.value || '').toLowerCase();
    
    // 渲染可选列表
    const filteredAvailable = availableTables.filter(item => 
        item.key.toLowerCase().includes(searchAvailable)
    );
    
    const availableList = document.getElementById('availableList');
    if (!availableList) return;
    
    if (filteredAvailable.length === 0) {
        availableList.innerHTML = '<div class="text-center text-muted p-4"><i class="bi bi-inbox"></i><div class="mt-2">没有可选项</div></div>';
    } else {
        // 按数据库分组
        const groupedAvailable = {};
        filteredAvailable.forEach(item => {
            if (!groupedAvailable[item.database]) {
                groupedAvailable[item.database] = [];
            }
            groupedAvailable[item.database].push(item);
        });
        
        availableList.innerHTML = Object.keys(groupedAvailable).sort().map(db => {
            const allSelected = groupedAvailable[db].every(item => 
                selectedTables.find(t => t.key === item.key)
            );
            
            return `
            <div class="transfer-group">
                <div class="transfer-group-header" style="display: flex; align-items: center;">
                    <label class="form-check mb-0 me-2">
                        <input type="checkbox" class="form-check-input" ${allSelected ? 'checked' : ''} onchange="toggleAvailableGroup('${db}', this.checked)" title="全选该库所有表">
                    </label>
                    <div style="flex: 1; cursor: pointer;" onclick="toggleAvailableDatabase('${db}')">
                        <i class="bi bi-chevron-right me-2 chevron-icon" id="chevron-available-${db}" style="transition: transform 0.3s; color: #667eea;"></i>
                        <i class="bi bi-database me-1" style="color: #667eea;"></i>
                        <strong>${db}</strong>
                        <span class="text-muted ms-2" style="font-size: 12px;">(${groupedAvailable[db].length})</span>
                    </div>
                </div>
                <div id="tables-available-${db}" style="display: none; padding-left: 40px;">
                    ${groupedAvailable[db].map(item => `
                        <div class="transfer-item">
                            <label class="form-check mb-0">
                                <input type="checkbox" class="form-check-input available-checkbox" data-key="${item.key}" data-database="${db}">
                                <span><i class="bi bi-table me-1" style="color: #764ba2;"></i>${item.table}</span>
                            </label>
                        </div>
                    `).join('')}
                </div>
            </div>
            `;
        }).join('');
    }
    
    document.getElementById('availableCount').textContent = filteredAvailable.length;
    
    // 渲染已选列表
    const filteredSelected = selectedTables.filter(item => 
        item.key.toLowerCase().includes(searchSelected)
    );
    
    const selectedList = document.getElementById('selectedList');
    if (!selectedList) return;
    
    if (filteredSelected.length === 0) {
        selectedList.innerHTML = '<div class="text-center text-muted p-4"><i class="bi bi-inbox"></i><div class="mt-2">暂无选择</div></div>';
    } else {
        // 按数据库分组
        const groupedSelected = {};
        filteredSelected.forEach(item => {
            if (!groupedSelected[item.database]) {
                groupedSelected[item.database] = [];
            }
            groupedSelected[item.database].push(item);
        });
        
        selectedList.innerHTML = Object.keys(groupedSelected).sort().map(db => {
            // 检查数据库名是否被修改
            const firstItem = groupedSelected[db][0];
            const isDatabaseModified = firstItem.isDatabaseModified || (firstItem.sourceDatabase && firstItem.sourceDatabase !== db);
            const databaseDisplayText = isDatabaseModified 
                ? `<span style="color: #3182ce; font-weight: 500;">${db}</span> <span style="color: #718096; font-size: 12px;">(原: ${firstItem.sourceDatabase || db})</span>`
                : db;
            
            return `
            <div class="transfer-group">
                <div class="transfer-group-header" style="display: flex; align-items: center;">
                    <label class="form-check mb-0 me-2">
                        <input type="checkbox" class="form-check-input" checked onchange="toggleSelectedGroup('${db}', this.checked)" title="取消选择该库所有表">
                    </label>
                    <div style="flex: 1; cursor: pointer;" onclick="toggleSelectedDatabase('${db}')">
                        <i class="bi bi-chevron-right me-2 chevron-icon" id="chevron-selected-${db}" style="transition: transform 0.3s; color: #667eea;"></i>
                        <i class="bi bi-database me-1" style="color: #667eea;"></i>
                        <strong>${databaseDisplayText}</strong>
                        <span class="text-muted ms-2" style="font-size: 12px;">(${groupedSelected[db].length})</span>
                    </div>
                </div>
                <div id="tables-selected-${db}" style="display: none; padding-left: 40px;">
                    ${groupedSelected[db].map(item => {
                        const isModified = item.isModified || item.table !== item.targetTable;
                        const displayText = isModified 
                            ? `<span style="color: #3182ce; font-weight: 500;">${item.targetTable}</span> <span style="color: #718096; font-size: 12px;">(原: ${item.table})</span>`
                            : item.table;
                        return `
                        <div class="transfer-item" style="display: flex; align-items: center; justify-content: space-between;">
                            <label class="form-check mb-0" style="flex: 1;">
                                <input type="checkbox" class="form-check-input selected-checkbox" data-key="${item.key}" data-database="${db}">
                                <span><i class="bi bi-table me-1" style="color: #764ba2;"></i>${displayText}</span>
                            </label>
                            <button type="button" class="btn btn-sm btn-link p-0 ms-2" onclick="showTableRenameModal('${item.key}')" title="修改表名" style="color: #667eea;">
                                <i class="bi bi-pencil"></i>
                            </button>
                        </div>
                        `;
                    }).join('')}
                </div>
            </div>
            `;
        }).join('');
    }
    
    document.getElementById('selectedCount').textContent = selectedTables.length;
}

// 切换可选列表数据库展开/折叠
window.toggleAvailableDatabase = function(database) {
    const tablesDiv = document.getElementById(`tables-available-${database}`);
    const chevron = document.getElementById(`chevron-available-${database}`);
    
    if (tablesDiv.style.display === 'none') {
        tablesDiv.style.display = 'block';
        chevron.style.transform = 'rotate(90deg)';
    } else {
        tablesDiv.style.display = 'none';
        chevron.style.transform = 'rotate(0deg)';
    }
};

// 切换已选列表数据库展开/折叠
window.toggleSelectedDatabase = function(database) {
    const tablesDiv = document.getElementById(`tables-selected-${database}`);
    const chevron = document.getElementById(`chevron-selected-${database}`);
    
    if (tablesDiv.style.display === 'none') {
        tablesDiv.style.display = 'block';
        chevron.style.transform = 'rotate(90deg)';
    } else {
        tablesDiv.style.display = 'none';
        chevron.style.transform = 'rotate(0deg)';
    }
};

// 全选/取消全选可选组
window.toggleAvailableGroup = function(database, checked) {
    if (checked) {
        const tablesToMove = availableTables.filter(item => item.database === database);
        tablesToMove.forEach(item => {
            if (!selectedTables.find(t => t.key === item.key)) {
                selectedTables.push({
                    ...item,
                    sourceDatabase: item.database,
                    targetTable: item.table,
                    isModified: false,
                    isDatabaseModified: false
                });
            }
        });
        renderTransferLists();
    } else {
        selectedTables = selectedTables.filter(item => item.database !== database);
        renderTransferLists();
    }
};

// 全选/取消全选已选组
window.toggleSelectedGroup = function(database, checked) {
    if (!checked) {
        selectedTables = selectedTables.filter(item => item.database !== database);
        renderTransferLists();
    }
};

// 移动选中项到已选
window.moveToSelected = function() {
    const checkboxes = document.querySelectorAll('.available-checkbox:checked');
    const keys = Array.from(checkboxes).map(cb => cb.dataset.key);
    
    if (keys.length === 0) {
        showToast('请先选择要添加的表', 'warning');
        return;
    }
    
    keys.forEach(key => {
        const item = availableTables.find(t => t.key === key);
        if (item && !selectedTables.find(t => t.key === key)) {
            selectedTables.push({
                ...item,
                sourceDatabase: item.database,
                targetTable: item.table,
                isModified: false,
                isDatabaseModified: false
            });
        }
    });
    
    renderTransferLists();
};

// 移动全部到已选
window.moveAllToSelected = function() {
    availableTables.forEach(item => {
        if (!selectedTables.find(t => t.key === item.key)) {
            selectedTables.push({
                ...item,
                sourceDatabase: item.database,
                targetTable: item.table,
                isModified: false,
                isDatabaseModified: false
            });
        }
    });
    renderTransferLists();
};

// 移动选中项到可选
window.moveToAvailable = function() {
    const checkboxes = document.querySelectorAll('.selected-checkbox:checked');
    const keys = Array.from(checkboxes).map(cb => cb.dataset.key);
    
    if (keys.length === 0) {
        showToast('请先选择要移除的表', 'warning');
        return;
    }
    
    selectedTables = selectedTables.filter(item => !keys.includes(item.key));
    renderTransferLists();
};

// 移动全部到可选
window.moveAllToAvailable = function() {
    selectedTables = [];
    renderTransferLists();
};

// 全选所有可选表
window.selectAllAvailable = function() {
    availableTables.forEach(item => {
        if (!selectedTables.find(t => t.key === item.key)) {
            selectedTables.push({
                ...item,
                sourceDatabase: item.database,
                targetTable: item.table,
                isModified: false,
                isDatabaseModified: false
            });
        }
    });
    renderTransferLists();
};

// 清空所有已选表
window.clearAllSelected = async function() {
    if (selectedTables.length === 0) {
        showToast('没有已选的表', 'info');
        return;
    }
    
    const confirmed = await showConfirmModal(`确定要清空所有已选的 ${selectedTables.length} 张表吗？`);
    if (confirmed) {
        selectedTables = [];
        renderTransferLists();
        showToast('已清空所有已选表', 'success');
    }
};
