// 任务监控模块入口文件
(function() {
    'use strict';
    
    window.TaskMonitorModule = {
        // 初始化模块
        init: function(taskId) {
            // 加载任务列表
            if (window.TaskMonitorList) {
                window.TaskMonitorList.load();
            }
            
            // 如果提供了 taskId，自动选择该任务
            if (taskId) {
                setTimeout(() => {
                    this.autoSelectTask(taskId);
                }, 500);
            }
        },
        
        // 自动选择任务
        autoSelectTask: function(taskId) {
            const taskItems = document.querySelectorAll('.task-item');
            taskItems.forEach(item => {
                if (item.getAttribute('onclick') && item.getAttribute('onclick').includes(taskId)) {
                    item.click();
                }
            });
        },
        
        // 清理模块
        destroy: function() {
            if (window.TaskMonitorSSE) {
                window.TaskMonitorSSE.closeAll();
            }
        }
    };
    
    // 页面卸载时清理
    window.addEventListener('beforeunload', function() {
        TaskMonitorModule.destroy();
    });
})();
