// Modal 对话框组件
(function() {
    'use strict';
    
    window.Modal = {
        // 确认对话框
        confirm: function(message, onConfirm, onCancel) {
            const modalId = 'confirmModal-' + Date.now();
            const modalHtml = `
                <div class="modal fade" id="${modalId}" tabindex="-1">
                    <div class="modal-dialog modal-dialog-centered">
                        <div class="modal-content">
                            <div class="modal-header">
                                <h5 class="modal-title">
                                    <i class="bi bi-question-circle text-warning me-2"></i>确认操作
                                </h5>
                                <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                            </div>
                            <div class="modal-body">${message}</div>
                            <div class="modal-footer">
                                <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">取消</button>
                                <button type="button" class="btn btn-primary" id="${modalId}-confirm">确定</button>
                            </div>
                        </div>
                    </div>
                </div>
            `;
            
            document.body.insertAdjacentHTML('beforeend', modalHtml);
            const modalElement = document.getElementById(modalId);
            const modal = new bootstrap.Modal(modalElement);
            
            document.getElementById(modalId + '-confirm').addEventListener('click', function() {
                modal.hide();
                if (onConfirm) onConfirm();
            });
            
            modalElement.addEventListener('hidden.bs.modal', function() {
                modalElement.remove();
                if (onCancel) onCancel();
            });
            
            modal.show();
        },
        
        // 输入对话框
        prompt: function(title, message, defaultValue, onConfirm, onCancel) {
            const modalId = 'promptModal-' + Date.now();
            const modalHtml = `
                <div class="modal fade" id="${modalId}" tabindex="-1">
                    <div class="modal-dialog modal-dialog-centered">
                        <div class="modal-content">
                            <div class="modal-header">
                                <h5 class="modal-title">
                                    <i class="bi bi-pencil-square text-primary me-2"></i>${title}
                                </h5>
                                <button type="button" class="btn-close" data-bs-dismiss="modal"></button>
                            </div>
                            <div class="modal-body">
                                <p class="mb-3">${message}</p>
                                <input type="text" class="form-control" id="${modalId}-input" value="${defaultValue || ''}" placeholder="请输入...">
                            </div>
                            <div class="modal-footer">
                                <button type="button" class="btn btn-secondary" data-bs-dismiss="modal">取消</button>
                                <button type="button" class="btn btn-primary" id="${modalId}-confirm">确定</button>
                            </div>
                        </div>
                    </div>
                </div>
            `;
            
            document.body.insertAdjacentHTML('beforeend', modalHtml);
            const modalElement = document.getElementById(modalId);
            const inputElement = document.getElementById(modalId + '-input');
            const modal = new bootstrap.Modal(modalElement);
            
            // 确定按钮
            document.getElementById(modalId + '-confirm').addEventListener('click', function() {
                const value = inputElement.value.trim();
                if (value) {
                    modal.hide();
                    if (onConfirm) onConfirm(value);
                } else {
                    inputElement.classList.add('is-invalid');
                    setTimeout(() => inputElement.classList.remove('is-invalid'), 2000);
                }
            });
            
            // 回车键确认
            inputElement.addEventListener('keypress', function(e) {
                if (e.key === 'Enter') {
                    document.getElementById(modalId + '-confirm').click();
                }
            });
            
            // 关闭时清理
            modalElement.addEventListener('hidden.bs.modal', function() {
                modalElement.remove();
            });
            
            modal.show();
            
            // 显示后自动聚焦并选中文本
            modalElement.addEventListener('shown.bs.modal', function() {
                inputElement.focus();
                inputElement.select();
            });
        }
    };
    
    // 兼容旧代码
    window.showConfirm = window.Modal.confirm.bind(window.Modal);
})();
