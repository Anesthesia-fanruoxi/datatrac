// 任务监控 - 侧边抽屉管理
(function() {
    'use strict';
    
    window.TaskMonitorDrawer = {
        currentDrawer: null,
        currentTaskId: null,
        
        // 打开进度抽屉
        openProgress: function(taskId, taskName) {
            this.closeAll();
            this.currentTaskId = taskId;
            this.currentDrawer = 'progress';
            
            this.createDrawer('progress', '任务进度 - ' + taskName, this.getProgressContent());
            
            // 启动进度SSE连接
            if (window.TaskMonitorProgressSSE) {
                window.TaskMonitorProgressSSE.start(taskId);
            }
            
            // 启动任务详情SSE（获取任务状态）
            if (window.TaskDetailSSE) {
                window.TaskDetailSSE.start(taskId);
            }
        },
        
        // 打开日志抽屉
        openLogs: function(taskId, taskName) {
            this.closeAll();
            this.currentTaskId = taskId;
            this.currentDrawer = 'logs';
            
            this.createDrawer('logs', '任务日志 - ' + taskName, this.getLogsContent());
            
            // 启动日志SSE连接
            if (window.TaskMonitorLogsSSE) {
                window.TaskMonitorLogsSSE.start(taskId, 'all');
            }
        },
        
        // 创建抽屉
        createDrawer: function(type, title, content) {
            // 移除已存在的抽屉
            const existingDrawer = document.querySelector('.task-drawer');
            if (existingDrawer) {
                existingDrawer.remove();
            }
            
            // 创建抽屉HTML
            const drawerHtml = `
                <div class="task-drawer" data-type="${type}">
                    <div class="drawer-overlay" onclick="TaskMonitorDrawer.close()"></div>
                    <div class="drawer-content">
                        <div class="drawer-header">
                            <h6><i class="bi bi-${type === 'progress' ? 'graph-up' : 'terminal'} me-2"></i>${title}</h6>
                            <button class="btn btn-sm btn-outline-secondary" onclick="TaskMonitorDrawer.close()">
                                <i class="bi bi-x"></i>
                            </button>
                        </div>
                        <div class="drawer-body">
                            ${content}
                        </div>
                    </div>
                </div>
            `;
            
            // 添加到页面
            document.body.insertAdjacentHTML('beforeend', drawerHtml);
            
            // 添加显示动画
            setTimeout(() => {
                const drawer = document.querySelector('.task-drawer');
                if (drawer) {
                    drawer.classList.add('show');
                }
            }, 10);
        },
        
        // 获取进度内容
        getProgressContent: function() {
            return `
                <div id="progressContent">
                    <div class="empty-state">
                        <i class="bi bi-hourglass-split"></i>
                        <div>加载进度数据中...</div>
                    </div>
                </div>
            `;
        },
        
        // 获取日志内容
        getLogsContent: function() {
            return `
                <div class="log-controls mb-3">
                    <div class="log-category-tabs">
                        <button class="log-category-tab active" onclick="TaskMonitorDrawer.switchLogCategory('all')">
                            全部
                        </button>
                        <button class="log-category-tab" onclick="TaskMonitorDrawer.switchLogCategory('initialize')">
                            初始化
                        </button>
                        <button class="log-category-tab" onclick="TaskMonitorDrawer.switchLogCategory('complete')">
                            完成
                        </button>
                    </div>
                </div>
                <div id="logContent">
                    <div class="empty-state">
                        <i class="bi bi-hourglass-split"></i>
                        <div>加载日志数据中...</div>
                    </div>
                </div>
            `;
        },
        
        // 切换日志分类
        switchLogCategory: function(category) {
            if (!this.currentTaskId) return;
            
            // 更新按钮状态
            document.querySelectorAll('.log-category-tab').forEach(tab => {
                tab.classList.remove('active');
            });
            event.currentTarget.classList.add('active');
            
            // 清空日志显示
            const logContent = document.getElementById('logContent');
            if (logContent) {
                logContent.innerHTML = `
                    <div class="empty-state">
                        <i class="bi bi-hourglass-split"></i>
                        <div>加载日志数据中...</div>
                    </div>
                `;
            }
            
            // 切换日志SSE连接
            if (window.TaskMonitorLogsSSE) {
                window.TaskMonitorLogsSSE.switchCategory(this.currentTaskId, category);
            }
        },
        
        // 关闭抽屉
        close: function() {
            const drawer = document.querySelector('.task-drawer');
            if (drawer) {
                drawer.classList.remove('show');
                setTimeout(() => {
                    drawer.remove();
                }, 300);
            }
            
            // 关闭所有SSE连接
            this.closeAllSSE();
            
            this.currentDrawer = null;
            this.currentTaskId = null;
        },
        
        // 关闭所有抽屉
        closeAll: function() {
            this.close();
        },
        
        // 关闭所有SSE连接
        closeAllSSE: function() {
            if (window.TaskMonitorSSE) {
                window.TaskMonitorSSE.closeAll();
            }
        },
        
        // 更新进度UI（由SSE调用）
        updateProgressUI: function(progress) {
            if (this.currentDrawer !== 'progress') return;
            
            const container = document.getElementById('progressContent');
            if (!container) return;
            
            // 使用原有的进度更新逻辑
            if (window.TaskMonitorDetail && window.TaskMonitorDetail.updateProgressUI) {
                window.TaskMonitorDetail.updateProgressUI(progress);
            }
        },
        
        // 添加日志（由SSE调用）
        appendLog: function(log) {
            if (this.currentDrawer !== 'logs') return;
            
            // 使用原有的日志添加逻辑
            if (window.TaskMonitorDetail && window.TaskMonitorDetail.appendLog) {
                window.TaskMonitorDetail.appendLog(log);
            }
        },
        
        // 清空日志
        clearLogs: function() {
            const logContent = document.getElementById('logContent');
            if (logContent) {
                logContent.innerHTML = '';
            }
        }
    };
    
    // 页面卸载时清理
    window.addEventListener('beforeunload', function() {
        TaskMonitorDrawer.closeAll();
    });
})();