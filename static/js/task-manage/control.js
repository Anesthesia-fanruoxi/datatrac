// 任务管理控制模块 - 启动、暂停、停止任务
(function() {
    'use strict';
    
    // 启动任务
    window.startTask = async function() {
        if (!window.TaskManageCore.currentTaskId) return;
        
        try {
            const response = await fetch(`/api/v1/tasks/${window.TaskManageCore.currentTaskId}/start`, {
                method: 'POST'
            });
            const result = await response.json();
            
            if (result.code === 200) {
                showToast('任务启动成功', 'success');
                await window.TaskManageCore.loadMonitorTasks();
                await window.TaskManageDetail.loadTaskDetail(window.TaskManageCore.currentTaskId);
                
                console.log('任务已启动，开始SSE连接');
                window.TaskManageCore.startSSE(window.TaskManageCore.currentTaskId);
            } else {
                showToast('启动失败: ' + result.message, 'error');
            }
        } catch (error) {
            showToast('启动失败: ' + error.message, 'error');
        }
    };

    // 暂停任务
    window.pauseTask = async function() {
        if (!window.TaskManageCore.currentTaskId) return;
        
        try {
            const response = await fetch(`/api/v1/tasks/${window.TaskManageCore.currentTaskId}/pause`, {
                method: 'POST'
            });
            const result = await response.json();
            
            if (result.code === 200) {
                showToast('任务暂停成功', 'success');
                window.TaskManageCore.closeSSE();
                console.log('任务已暂停，关闭SSE连接');
                await window.TaskManageCore.loadMonitorTasks();
                await window.TaskManageDetail.loadTaskDetail(window.TaskManageCore.currentTaskId);
            } else {
                showToast('暂停失败: ' + result.message, 'error');
            }
        } catch (error) {
            showToast('暂停失败: ' + error.message, 'error');
        }
    };

    // 显示停止任务确认模态框
    function showStopTaskModal() {
        // 创建模态框元素
        const modal = document.createElement('div');
        modal.id = 'stopTaskModal';
        modal.style.position = 'fixed';
        modal.style.top = '0';
        modal.style.left = '0';
        modal.style.width = '100%';
        modal.style.height = '100%';
        modal.style.backgroundColor = 'rgba(0, 0, 0, 0.5)';
        modal.style.display = 'flex';
        modal.style.alignItems = 'center';
        modal.style.justifyContent = 'center';
        modal.style.zIndex = '10000';
        
        // 创建模态框内容
        const modalContent = document.createElement('div');
        modalContent.style.backgroundColor = '#ffffff';
        modalContent.style.borderRadius = '12px';
        modalContent.style.padding = '30px';
        modalContent.style.boxShadow = '0 10px 30px rgba(0, 0, 0, 0.3)';
        modalContent.style.width = '400px';
        modalContent.style.maxWidth = '90%';
        
        // 添加标题
        const title = document.createElement('h4');
        title.textContent = '确认停止任务';
        title.style.fontSize = '18px';
        title.style.fontWeight = '600';
        title.style.color = '#2d3748';
        title.style.marginTop = '0';
        title.style.marginBottom = '16px';
        
        // 添加消息
        const message = document.createElement('p');
        message.textContent = '确定要停止任务吗？停止后将清除运行记录。';
        message.style.fontSize = '14px';
        message.style.color = '#4a5568';
        message.style.marginBottom = '24px';
        message.style.lineHeight = '1.5';
        
        // 添加按钮容器
        const buttonContainer = document.createElement('div');
        buttonContainer.style.display = 'flex';
        buttonContainer.style.justifyContent = 'flex-end';
        buttonContainer.style.gap = '12px';
        
        // 添加取消按钮
        const cancelButton = document.createElement('button');
        cancelButton.textContent = '取消';
        cancelButton.style.padding = '8px 16px';
        cancelButton.style.border = '1px solid #e2e8f0';
        cancelButton.style.borderRadius = '6px';
        cancelButton.style.backgroundColor = '#ffffff';
        cancelButton.style.color = '#4a5568';
        cancelButton.style.fontSize = '14px';
        cancelButton.style.cursor = 'pointer';
        cancelButton.style.transition = 'all 0.2s ease';
        
        cancelButton.addEventListener('mouseenter', function() {
            cancelButton.style.backgroundColor = '#f7fafc';
        });
        
        cancelButton.addEventListener('mouseleave', function() {
            cancelButton.style.backgroundColor = '#ffffff';
        });
        
        cancelButton.addEventListener('click', function() {
            document.body.removeChild(modal);
        });
        
        // 添加确定按钮
        const confirmButton = document.createElement('button');
        confirmButton.textContent = '确定';
        confirmButton.style.padding = '8px 16px';
        confirmButton.style.border = 'none';
        confirmButton.style.borderRadius = '6px';
        confirmButton.style.backgroundColor = '#3182ce';
        confirmButton.style.color = '#ffffff';
        confirmButton.style.fontSize = '14px';
        confirmButton.style.cursor = 'pointer';
        confirmButton.style.transition = 'all 0.2s ease';
        
        confirmButton.addEventListener('mouseenter', function() {
            confirmButton.style.backgroundColor = '#2c5282';
        });
        
        confirmButton.addEventListener('mouseleave', function() {
            confirmButton.style.backgroundColor = '#3182ce';
        });
        
        confirmButton.addEventListener('click', async function() {
            document.body.removeChild(modal);
            
            try {
                const response = await fetch(`/api/v1/tasks/${window.TaskManageCore.currentTaskId}/stop`, {
                    method: 'POST'
                });
                const result = await response.json();
                
                if (result.code === 200) {
                    showToast('任务停止成功', 'success');
                    window.TaskManageCore.closeSSE();
                    console.log('任务已停止，关闭SSE连接');
                    await window.TaskManageCore.loadMonitorTasks();
                    await window.TaskManageDetail.loadTaskDetail(window.TaskManageCore.currentTaskId);
                } else {
                    showToast('停止失败: ' + result.message, 'error');
                }
            } catch (error) {
                showToast('停止失败: ' + error.message, 'error');
            }
        });
        
        // 组装模态框
        buttonContainer.appendChild(cancelButton);
        buttonContainer.appendChild(confirmButton);
        
        modalContent.appendChild(title);
        modalContent.appendChild(message);
        modalContent.appendChild(buttonContainer);
        
        modal.appendChild(modalContent);
        
        // 添加到页面
        document.body.appendChild(modal);
        
        // 点击模态框外部关闭
        modal.addEventListener('click', function(e) {
            if (e.target === modal) {
                document.body.removeChild(modal);
            }
        });
    }

    // 停止任务
    window.stopTask = function() {
        if (!window.TaskManageCore.currentTaskId) return;
        
        showStopTaskModal();
    };

})();
