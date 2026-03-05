// HTTP 请求封装
(function() {
    'use strict';
    
    window.HttpUtils = {
        // GET 请求
        get: async function(url) {
            try {
                const response = await fetch(url);
                return await response.json();
            } catch (error) {
                console.error('GET 请求失败:', error);
                throw error;
            }
        },
        
        // POST 请求
        post: async function(url, data) {
            try {
                const response = await fetch(url, {
                    method: 'POST',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(data)
                });
                return await response.json();
            } catch (error) {
                console.error('POST 请求失败:', error);
                throw error;
            }
        },
        
        // PUT 请求
        put: async function(url, data) {
            try {
                const response = await fetch(url, {
                    method: 'PUT',
                    headers: {'Content-Type': 'application/json'},
                    body: JSON.stringify(data)
                });
                return await response.json();
            } catch (error) {
                console.error('PUT 请求失败:', error);
                throw error;
            }
        },
        
        // DELETE 请求
        delete: async function(url) {
            try {
                const response = await fetch(url, {
                    method: 'DELETE'
                });
                return await response.json();
            } catch (error) {
                console.error('DELETE 请求失败:', error);
                throw error;
            }
        }
    };
})();
