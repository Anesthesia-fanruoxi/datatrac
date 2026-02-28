// 任务配置向导 - 模态框管理

// 显示确认对话框
function showConfirmModal(message) {
    return new Promise((resolve) => {
        document.getElementById('confirmMessage').textContent = message;
        document.getElementById('confirmModal').style.display = 'block';
        
        // 保存回调函数
        window._confirmModalResolve = resolve;
    });
}

// 关闭确认对话框
window.closeConfirmModal = function(confirmed) {
    document.getElementById('confirmModal').style.display = 'none';
    if (window._confirmModalResolve) {
        window._confirmModalResolve(confirmed);
        window._confirmModalResolve = null;
    }
};

// 显示数据库名替换模态框
window.showDatabaseRenameModal = function() {
    if (selectedTables.length === 0) {
        showToast('请先选择要同步的表', 'warning');
        return;
    }
    
    // 重置表单
    document.getElementById('renameOldText').value = '';
    document.getElementById('renameNewText').value = '';
    document.querySelector('input[name="renameType"][value="prefix"]').checked = true;
    
    document.getElementById('databaseRenameModal').style.display = 'block';
};

// 关闭数据库名替换模态框
window.closeDatabaseRenameModal = function() {
    document.getElementById('databaseRenameModal').style.display = 'none';
};

// 应用数据库名替换
window.applyDatabaseRename = function() {
    const renameType = document.querySelector('input[name="renameType"]:checked').value;
    const oldText = document.getElementById('renameOldText').value;
    const newText = document.getElementById('renameNewText').value;
    
    if (!oldText) {
        showToast('请输入要替换的文本', 'warning');
        return;
    }
    
    // 统计被替换的数据库数量
    const replacedDatabases = new Set();
    
    // 遍历已选表，替换数据库名
    selectedTables.forEach(item => {
        const originalDatabase = item.sourceDatabase || item.database;
        let newDatabase = item.database;
        let replaced = false;
        
        if (renameType === 'prefix') {
            // 替换前缀
            if (newDatabase.startsWith(oldText)) {
                newDatabase = newText + newDatabase.substring(oldText.length);
                replaced = true;
            }
        } else {
            // 替换后缀
            if (newDatabase.endsWith(oldText)) {
                newDatabase = newDatabase.substring(0, newDatabase.length - oldText.length) + newText;
                replaced = true;
            }
        }
        
        if (replaced) {
            replacedDatabases.add(item.database); // 记录原数据库名
            
            // 更新数据库名，保持其他字段不变
            item.database = newDatabase;
            item.sourceDatabase = originalDatabase; // 保存原始数据库名
            item.isDatabaseModified = true;
            item.key = `${newDatabase}.${item.table}`;
        }
    });
    
    if (replacedDatabases.size === 0) {
        showToast('没有匹配的数据库名', 'warning');
        return;
    }
    
    // 重新渲染
    renderTransferLists();
    
    // 关闭模态框
    closeDatabaseRenameModal();
    
    showToast(`成功替换 ${replacedDatabases.size} 个数据库名，共 ${selectedTables.length} 张表`, 'success');
};

// 显示单个表名修改模态框
window.showTableRenameModal = function(tableKey) {
    const table = selectedTables.find(t => t.key === tableKey);
    if (!table) {
        showToast('未找到该表', 'error');
        return;
    }
    
    // 保存当前编辑的表
    window._currentEditingTableKey = tableKey;
    
    // 设置表单值
    document.getElementById('tableRenameSourceName').textContent = `${table.database}.${table.table}`;
    document.getElementById('tableRenameTargetName').value = table.targetTable || table.table;
    
    document.getElementById('tableRenameModal').style.display = 'block';
};

// 关闭表名修改模态框
window.closeTableRenameModal = function() {
    document.getElementById('tableRenameModal').style.display = 'none';
    window._currentEditingTableKey = null;
};

// 应用单个表名修改
window.applyTableRename = function() {
    const newTableName = document.getElementById('tableRenameTargetName').value.trim();
    
    if (!newTableName) {
        showToast('表名不能为空', 'warning');
        return;
    }
    
    const tableKey = window._currentEditingTableKey;
    const table = selectedTables.find(t => t.key === tableKey);
    
    if (!table) {
        showToast('未找到该表', 'error');
        return;
    }
    
    // 更新表名
    table.targetTable = newTableName;
    table.isModified = (newTableName !== table.table);
    
    // 重新渲染
    renderTransferLists();
    
    // 关闭模态框
    closeTableRenameModal();
    
    showToast('表名修改成功', 'success');
};

// 显示批量修改表名模态框
window.showBatchTableRenameModal = function() {
    if (selectedTables.length === 0) {
        showToast('请先选择要同步的表', 'warning');
        return;
    }
    
    // 重置表单
    document.getElementById('batchTableRenameOldText').value = '';
    document.getElementById('batchTableRenameNewText').value = '';
    document.querySelector('input[name="batchTableRenameType"][value="add_prefix"]').checked = true;
    
    document.getElementById('batchTableRenameModal').style.display = 'block';
};

// 关闭批量修改表名模态框
window.closeBatchTableRenameModal = function() {
    document.getElementById('batchTableRenameModal').style.display = 'none';
};

// 应用批量修改表名
window.applyBatchTableRename = function() {
    const renameType = document.querySelector('input[name="batchTableRenameType"]:checked').value;
    const oldText = document.getElementById('batchTableRenameOldText').value;
    const newText = document.getElementById('batchTableRenameNewText').value;
    
    let modifiedCount = 0;
    
    selectedTables.forEach(item => {
        let newTableName = item.targetTable || item.table;
        let modified = false;
        
        if (renameType === 'add_prefix') {
            // 添加前缀
            if (newText) {
                newTableName = newText + newTableName;
                modified = true;
            }
        } else if (renameType === 'add_suffix') {
            // 添加后缀
            if (newText) {
                newTableName = newTableName + newText;
                modified = true;
            }
        } else if (renameType === 'replace_prefix') {
            // 替换前缀
            if (oldText && newTableName.startsWith(oldText)) {
                newTableName = newText + newTableName.substring(oldText.length);
                modified = true;
            }
        } else if (renameType === 'replace_suffix') {
            // 替换后缀
            if (oldText && newTableName.endsWith(oldText)) {
                newTableName = newTableName.substring(0, newTableName.length - oldText.length) + newText;
                modified = true;
            }
        }
        
        if (modified) {
            item.targetTable = newTableName;
            item.isModified = (newTableName !== item.table);
            modifiedCount++;
        }
    });
    
    if (modifiedCount === 0) {
        showToast('没有表名被修改', 'warning');
        return;
    }
    
    // 重新渲染
    renderTransferLists();
    
    // 关闭模态框
    closeBatchTableRenameModal();
    
    showToast(`成功修改 ${modifiedCount} 个表名`, 'success');
};
