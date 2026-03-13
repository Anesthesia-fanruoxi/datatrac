// 数据源 SSE 连接管理
(function() {
    'use strict';
    
    window.DataSourceSSE = {
        eventSource: null,
        
        // 启动 SSE 连接
        start: function() {
            this.close();
            
            this.eventSource = new EventSource('/api/v1/datasources/test/stream');
            
            // 监听测试事件（后端推送数组）
            this.eventSource.addEventListener('test', (e) => {
                try {
                    const results = JSON.parse(e.data);
                    // 只处理数组格式
                    if (Array.isArray(results)) {
                        results.forEach(item => this.updateStatus(item));
                    }
                } catch (error) {
                    console.error('解析测试结果失败:', error);
                }
            });
            
            // 监听错误
            this.eventSource.onerror = (e) => {
                console.error('数据源 SSE 连接错误:', e);
            };
        },
        
        // 关闭 SSE 连接
        close: function() {
            if (this.eventSource) {
                this.eventSource.close();
                this.eventSource = null;
            }
        },
        
        // 更新状态
        updateStatus: function(result) {
            if (!result || !result.id) return;
            const statusCell = document.getElementById(`status-${result.id}`);
            if (!statusCell) return;
            
            if (result.status === 'testing') {
                statusCell.innerHTML = `
                    <span class="text-muted">
                        <i class="bi bi-hourglass-split"></i> 检测中...
                    </span>
                `;
            } else if (result.status === 'success') {
                statusCell.innerHTML = `
                    <span class="text-success" title="${result.message}">
                        <i class="bi bi-check-circle-fill"></i> 正常
                    </span>
                `;
            } else {
                statusCell.innerHTML = `
                    <span class="text-danger" title="${result.message}">
                        <i class="bi bi-x-circle-fill"></i> 失败
                    </span>
                `;
            }
        }
    };
})();
