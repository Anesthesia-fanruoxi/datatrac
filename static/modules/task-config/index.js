// 任务配置模块入口文件
(function() {
    'use strict';
    
    window.TaskConfigModule = {
        // 初始化模块
        init: function() {
            console.log('任务配置模块初始化');
            
            // 加载任务列表
            TaskConfigCore.loadList();
            
            // 绑定事件
            this.bindEvents();
        },
        
        // 绑定事件
        bindEvents: function() {
            // 新建任务表单提交
            const form = document.getElementById('createTaskForm');
            if (form) {
                form.addEventListener('submit', (e) => TaskConfigCore.submitCreate(e));
            }
        },
        
        // 清理模块
        destroy: function() {
            console.log('任务配置模块清理');
        }
    };
    
    // 页面卸载时清理
    window.addEventListener('beforeunload', function() {
        TaskConfigModule.destroy();
    });
})();
