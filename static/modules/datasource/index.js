// 数据源模块入口文件
(function() {
    'use strict';
    
    window.DataSourceModule = {
        // 初始化模块
        init: function() {
            // 加载数据源列表
            DataSourceCore.loadList();
            
            // 绑定事件
            this.bindEvents();
        },
        
        // 绑定事件
        bindEvents: function() {
            // 表单提交
            const form = document.getElementById('datasourceForm');
            if (form) {
                form.addEventListener('submit', (e) => DataSourceCore.submitForm(e));
            }
            
            // 类型切换
            const typeSelect = document.getElementById('dsType');
            if (typeSelect) {
                typeSelect.addEventListener('change', function() {
                    const dbNameGroup = document.getElementById('dbNameGroup');
                    dbNameGroup.style.display = this.value === 'mysql' ? 'block' : 'none';
                });
            }
        },
        
        // 清理模块
        destroy: function() {
            if (window.DataSourceSSE) {
                window.DataSourceSSE.close();
            }
        }
    };
    
    // 页面卸载时清理
    window.addEventListener('beforeunload', function() {
        DataSourceModule.destroy();
    });
})();
