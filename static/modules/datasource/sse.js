// 数据源 SSE 连接管理
(function() {
    'use strict';
    
    window.DataSourceSSE = {
        eventSource: null,
        
        // 启动 SSE 连接
        start: function() {
            this.close();
            
            console.log('启动数据源 SSE 测试');
            this.eventSource = new EventSource('/api/v1/datasources/test/stream');
            
            // 监听测试事件
            this.eventSource.addEventListener('test', (e) => {
                try {
                    const result = JSON.parse(e.data);
                    console.log('收到测试结果:', result);
                    this.updateStatus(result);
                } catch (error) {
                    console.error('解析测试结果失败:', error);
                }
            });
            
            // 监听错误
            this.eventSource.onerror = (e) => {
                console.error('数据源 SSE 连接错误:', e);
            };
            
            // 监听连接打开
            this.eventSource.onopen = () => {
                console.log('数据源 SSE 连接已建立');
            };
        },
        
        // 关闭 SSE 连接
        close: function() {
            if (this.eventSource) {
                console.log('关闭数据源 SSE 连接');
                this.eventSource.close();
                this.eventSource = null;
            }
        },
        
        // 更新状态
        updateStatus: function(result) {
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
