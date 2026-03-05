// 任务监控 - 日志 SSE 连接
(function() {
    'use strict';
    
    window.TaskMonitorLogsSSE = {
        eventSource: null,
        currentCategory: 'all',
        currentTaskId: null,
        
        // 启动日志 SSE 连接（支持category参数）
        start: function(taskId, category) {
            category = category || 'all';
            
            // 如果category没变，不需要重新连接
            if (this.eventSource && this.currentTaskId === taskId && this.currentCategory === category) {
                console.log('日志 SSE 已连接，category相同，无需重新连接');
                return;
            }
            
            // 关闭旧连接
            this.close();
            
            this.currentTaskId = taskId;
            this.currentCategory = category;
            
            console.log('启动日志 SSE 连接，任务ID:', taskId, 'category:', category);
            this.eventSource = new EventSource(`/api/v1/tasks/${taskId}/stream/logs?category=${category}`);
            
            // 监听日志事件
            this.eventSource.addEventListener('log', (e) => {
                try {
                    const logs = JSON.parse(e.data);
                    console.log('收到日志（数组）:', logs);
                    
                    // 后端推送的是日志数组，需要遍历追加
                    if (window.TaskMonitorDetail && Array.isArray(logs)) {
                        logs.forEach(log => {
                            window.TaskMonitorDetail.appendLog(log);
                        });
                    }
                } catch (error) {
                    console.error('解析日志数据失败:', error);
                }
            });
            
            // 监听错误
            this.eventSource.onerror = (e) => {
                console.error('日志 SSE 连接错误:', e);
                // SSE 会自动重连，这里只记录错误
            };
            
            // 监听连接打开
            this.eventSource.onopen = () => {
                console.log('日志 SSE 连接已建立');
            };
        },
        
        // 切换日志分类
        switchCategory: function(taskId, category) {
            console.log('切换日志分类:', category);
            
            // 清空日志显示
            if (window.TaskMonitorDetail) {
                window.TaskMonitorDetail.clearLogs();
            }
            
            // 重新连接SSE（会自动关闭旧连接）
            this.start(taskId, category);
        },
        
        // 关闭 SSE 连接
        close: function() {
            if (this.eventSource) {
                console.log('关闭日志 SSE 连接');
                this.eventSource.close();
                this.eventSource = null;
                this.currentTaskId = null;
                this.currentCategory = 'all';
            }
        }
    };
    
    // SSE 管理器
    window.TaskMonitorSSE = {
        closeAll: function() {
            console.log('关闭所有 SSE 连接');
            if (window.TaskDetailSSE) {
                window.TaskDetailSSE.close();
            }
            if (window.TaskMonitorProgressSSE) {
                window.TaskMonitorProgressSSE.close();
            }
            if (window.TaskMonitorLogsSSE) {
                window.TaskMonitorLogsSSE.close();
            }
        }
    };
})();
