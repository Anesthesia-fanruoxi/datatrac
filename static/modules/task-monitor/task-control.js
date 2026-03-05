// 任务监控 - 任务控制
(function() {
    'use strict';
    
    window.TaskMonitorControl = {
        // 启动任务
        start: async function() {
            const taskId = window.TaskMonitorList?.currentTaskId;
            if (!taskId) {
                Toast.warning('请先选择任务');
                return;
            }
            
            try {
                const result = await HttpUtils.post(`/api/v1/tasks/${taskId}/start`, {});
                
                if (result.code === 200) {
                    Toast.success('任务启动成功！');
                    if (window.TaskMonitorList) {
                        window.TaskMonitorList.load();
                    }
                } else {
                    Toast.error('启动失败: ' + result.message);
                }
            } catch (error) {
                Toast.error('启动失败: ' + error.message);
            }
        },
        
        // 暂停任务
        pause: async function() {
            const taskId = window.TaskMonitorList?.currentTaskId;
            if (!taskId) {
                Toast.warning('请先选择任务');
                return;
            }
            
            try {
                const result = await HttpUtils.post(`/api/v1/tasks/${taskId}/pause`, {});
                
                if (result.code === 200) {
                    Toast.success('任务已暂停！');
                    if (window.TaskMonitorList) {
                        window.TaskMonitorList.load();
                    }
                } else {
                    Toast.error('暂停失败: ' + result.message);
                }
            } catch (error) {
                Toast.error('暂停失败: ' + error.message);
            }
        },
        
        // 停止任务
        stop: async function() {
            const taskId = window.TaskMonitorList?.currentTaskId;
            if (!taskId) {
                Toast.warning('请先选择任务');
                return;
            }
            
            Modal.confirm('确定要停止任务吗？', async () => {
                try {
                    const result = await HttpUtils.post(`/api/v1/tasks/${taskId}/stop`, {});
                    
                    if (result.code === 200) {
                        Toast.success('任务已停止！');
                        if (window.TaskMonitorList) {
                            window.TaskMonitorList.load();
                        }
                    } else {
                        Toast.error('停止失败: ' + result.message);
                    }
                } catch (error) {
                    Toast.error('停止失败: ' + error.message);
                }
            });
        }
    };
    
    // 全局函数（供 HTML onclick 调用）
    window.startTask = function() {
        TaskMonitorControl.start();
    };
    
    window.pauseTask = function() {
        TaskMonitorControl.pause();
    };
    
    window.stopTask = function() {
        TaskMonitorControl.stop();
    };
})();
