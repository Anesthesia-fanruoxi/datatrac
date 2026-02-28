// 通用工具函数

// 格式化日期
function formatDate(dateString) {
    if (!dateString) return '-';
    const date = new Date(dateString);
    return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit'
    });
}

// 格式化文件大小
function formatSize(bytes) {
    if (bytes === 0) return '0 B';
    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
}

// 格式化数字（添加千分位）
function formatNumber(num) {
    return num.toString().replace(/\B(?=(\d{3})+(?!\d))/g, ',');
}

// 显示加载提示
function showLoading(message = '加载中...') {
    // 可以使用 Bootstrap 的 Modal 或 Toast
    console.log(message);
}

// 隐藏加载提示
function hideLoading() {
    console.log('加载完成');
}

// 显示成功提示
function showSuccess(message) {
    alert(message);
}

// 显示错误提示
function showError(message) {
    alert('错误: ' + message);
}

// 确认对话框
function confirm(message) {
    return window.confirm(message);
}
