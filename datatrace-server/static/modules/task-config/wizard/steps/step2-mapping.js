// 任务配置向导 - 步骤2：映射管理模块
(function() {
    'use strict';
    
    window.TaskWizardStep2Mapping = {
        // 数据结构：{ database: { source_database, target_database, is_database_modified, tables: [{source_table, target_table, is_modified}] } }
        mappings: {},
        
        // 从 selected_databases 配置加载映射
        loadFromConfig: function(selectedDatabases) {
            this.mappings = {};
            
            if (!selectedDatabases || !Array.isArray(selectedDatabases)) {
                return;
            }
            
            selectedDatabases.forEach(db => {
                const dbKey = db.database || db.source_database;
                this.mappings[dbKey] = {
                    source_database: db.source_database || db.database,
                    target_database: db.target_database || db.database,
                    is_database_modified: db.is_database_modified || false,
                    tables: (db.tables || []).map(t => ({
                        source_table: t.source_table,
                        target_table: t.target_table || t.source_table,
                        is_modified: t.is_modified || false,
                        selected_fields: t.selected_fields || []
                    }))
                };
            });
        },
        
        // 从 selected_tables 添加映射（默认源=目标）
        addFromSelectedTables: function(selectedTables) {
            Object.keys(selectedTables).forEach(dbName => {
                if (!this.mappings[dbName]) {
                    this.mappings[dbName] = {
                        source_database: dbName,
                        target_database: dbName,
                        is_database_modified: false,
                        tables: []
                    };
                }
                
                const tables = selectedTables[dbName] || [];
                tables.forEach(tableName => {
                    // 检查是否已存在
                    const exists = this.mappings[dbName].tables.find(t => t.source_table === tableName);
                    if (!exists) {
                        this.mappings[dbName].tables.push({
                            source_table: tableName,
                            target_table: tableName,
                            is_modified: false,
                            selected_fields: []
                        });
                    }
                });
            });
        },
        
        // 移除表映射
        removeTable: function(dbName, tableName) {
            if (this.mappings[dbName]) {
                this.mappings[dbName].tables = this.mappings[dbName].tables.filter(
                    t => t.source_table !== tableName
                );
                
                // 如果数据库下没有表了，删除整个数据库映射
                if (this.mappings[dbName].tables.length === 0) {
                    delete this.mappings[dbName];
                }
            }
        },
        
        // 修改数据库名映射
        updateDatabaseMapping: function(sourceDb, targetDb) {
            if (this.mappings[sourceDb]) {
                this.mappings[sourceDb].target_database = targetDb;
                this.mappings[sourceDb].is_database_modified = (sourceDb !== targetDb);
            }
        },
        
        // 修改表名映射
        updateTableMapping: function(dbName, sourceTable, targetTable) {
            if (this.mappings[dbName]) {
                const table = this.mappings[dbName].tables.find(t => t.source_table === sourceTable);
                if (table) {
                    table.target_table = targetTable;
                    table.is_modified = (sourceTable !== targetTable);
                }
            }
        },
        
        // 批量操作：添加前缀
        batchAddPrefix: function(dbName, tables, prefix) {
            if (!this.mappings[dbName]) return;
            
            tables.forEach(tableName => {
                const table = this.mappings[dbName].tables.find(t => t.source_table === tableName);
                if (table) {
                    table.target_table = prefix + table.source_table;
                    table.is_modified = true;
                }
            });
        },
        
        // 批量操作：添加后缀
        batchAddSuffix: function(dbName, tables, suffix) {
            if (!this.mappings[dbName]) return;
            
            tables.forEach(tableName => {
                const table = this.mappings[dbName].tables.find(t => t.source_table === tableName);
                if (table) {
                    table.target_table = table.source_table + suffix;
                    table.is_modified = true;
                }
            });
        },
        
        // 批量操作：替换前缀
        batchReplacePrefix: function(dbName, tables, oldPrefix, newPrefix) {
            if (!this.mappings[dbName]) return;
            
            tables.forEach(tableName => {
                const table = this.mappings[dbName].tables.find(t => t.source_table === tableName);
                if (table && table.source_table.startsWith(oldPrefix)) {
                    table.target_table = newPrefix + table.source_table.substring(oldPrefix.length);
                    table.is_modified = true;
                }
            });
        },
        
        // 批量操作：替换后缀
        batchReplaceSuffix: function(dbName, tables, oldSuffix, newSuffix) {
            if (!this.mappings[dbName]) return;
            
            tables.forEach(tableName => {
                const table = this.mappings[dbName].tables.find(t => t.source_table === tableName);
                if (table && table.source_table.endsWith(oldSuffix)) {
                    table.target_table = table.source_table.substring(0, table.source_table.length - oldSuffix.length) + newSuffix;
                    table.is_modified = true;
                }
            });
        },
        
        // 重置映射（源=目标）
        resetMapping: function(dbName, tableName) {
            if (tableName) {
                // 重置单个表
                if (this.mappings[dbName]) {
                    const table = this.mappings[dbName].tables.find(t => t.source_table === tableName);
                    if (table) {
                        table.target_table = table.source_table;
                        table.is_modified = false;
                    }
                }
            } else {
                // 重置整个数据库
                if (this.mappings[dbName]) {
                    this.mappings[dbName].target_database = this.mappings[dbName].source_database;
                    this.mappings[dbName].is_database_modified = false;
                    this.mappings[dbName].tables.forEach(table => {
                        table.target_table = table.source_table;
                        table.is_modified = false;
                    });
                }
            }
        },
        
        // 转换为配置格式
        toConfig: function() {
            return Object.keys(this.mappings).map(dbName => {
                const mapping = this.mappings[dbName];
                return {
                    database: mapping.target_database,
                    source_database: mapping.source_database,
                    is_database_modified: mapping.is_database_modified,
                    tables: mapping.tables.map(t => ({
                        source_table: t.source_table,
                        target_table: t.target_table,
                        is_modified: t.is_modified,
                        selected_fields: t.selected_fields || []
                    }))
                };
            });
        },
        
        // 更新表的字段配置
        updateTableFields: function(dbName, tableName, selectedFields) {
            if (this.mappings[dbName]) {
                const table = this.mappings[dbName].tables.find(t => t.source_table === tableName);
                if (table) {
                    table.selected_fields = selectedFields || [];
                }
            }
        },
        
        // 获取已选择的表（用于左侧回显）
        getSelectedTables: function() {
            const result = {};
            Object.keys(this.mappings).forEach(dbName => {
                result[dbName] = this.mappings[dbName].tables.map(t => t.source_table);
            });
            return result;
        }
    };
})();
