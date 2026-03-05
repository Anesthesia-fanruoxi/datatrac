// 任务配置向导核心逻辑
(function() {
    'use strict';
    
    window.TaskWizard = {
        currentTaskId: null,
        currentStep: 1,
        totalSteps: 4,
        taskData: {},
        
        // 显示配置向导
        show: async function(taskId) {
            this.currentTaskId = taskId;
            this.currentStep = 1;
            this.taskData = {};
            
            // 创建向导模态框
            this.createModal();
            
            // 加载任务信息（等待完成）
            const success = await this.loadTaskInfo();
            
            if (!success) {
                Toast.error('加载任务信息失败');
                this.close();
                return;
            }
            
            // 显示第一步
            this.showStep(1);
        },
        
        // 创建向导模态框
        createModal: function() {
            const modalHtml = `
            <div id="wizardModal" style="display:block; position:fixed; top:0; left:0; width:100%; height:100%; background:rgba(0,0,0,0.6); z-index:1001; backdrop-filter: blur(5px);">
                <div style="position:absolute; top:50%; left:50%; transform:translate(-50%,-50%); background:white; padding:0; border-radius:15px; width:90%; max-width:1200px; height:85vh; display: flex; flex-direction: column; box-shadow: 0 20px 60px rgba(0,0,0,0.3);">
                    <!-- 头部 -->
                    <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 20px; border-radius: 15px 15px 0 0; flex-shrink: 0;">
                        <h5 style="color: white; margin: 0;"><i class="bi bi-gear me-2"></i>任务配置向导</h5>
                        <div style="color: rgba(255,255,255,0.9); font-size: 14px; margin-top: 5px;" id="wizardTaskInfo">
                            加载中...
                        </div>
                    </div>
                    
                    <!-- 步骤指示器 -->
                    <div style="padding: 20px; border-bottom: 1px solid #e2e8f0; flex-shrink: 0;">
                        <div class="wizard-steps" id="wizardSteps">
                            <div class="wizard-step active" data-step="1" onclick="TaskWizard.goToStep(1)">
                                <div class="step-number">1</div>
                                <div class="step-title">选择数据源</div>
                            </div>
                            <div class="wizard-step" data-step="2" onclick="TaskWizard.goToStep(2)">
                                <div class="step-number">2</div>
                                <div class="step-title">选择表</div>
                            </div>
                            <div class="wizard-step" data-step="3" onclick="TaskWizard.goToStep(3)">
                                <div class="step-number">3</div>
                                <div class="step-title">同步配置</div>
                            </div>
                            <div class="wizard-step" data-step="4" onclick="TaskWizard.goToStep(4)">
                                <div class="step-number">4</div>
                                <div class="step-title">确认配置</div>
                            </div>
                        </div>
                    </div>
                    
                    <!-- 内容区 -->
                    <div style="flex: 1; overflow-y: auto; padding: 30px;" id="wizardContent">
                        <!-- 步骤内容将在这里动态加载 -->
                    </div>
                    
                    <!-- 底部按钮 -->
                    <div style="padding: 20px; border-top: 1px solid #e2e8f0; background: #f7fafc; border-radius: 0 0 15px 15px; flex-shrink: 0;">
                        <div class="d-flex justify-content-between">
                            <button type="button" class="btn btn-secondary" onclick="TaskWizard.close()">取消</button>
                            <div>
                                <button type="button" class="btn btn-outline-primary me-2" id="wizardPrevBtn" onclick="TaskWizard.prevStep()" style="display:none;">
                                    <i class="bi bi-arrow-left"></i> 上一步
                                </button>
                                <button type="button" class="btn btn-primary" id="wizardNextBtn" onclick="TaskWizard.nextStep()">
                                    下一步 <i class="bi bi-arrow-right"></i>
                                </button>
                                <button type="button" class="btn btn-success" id="wizardFinishBtn" onclick="TaskWizard.finish()" style="display:none;">
                                    <i class="bi bi-check-circle"></i> 完成配置
                                </button>
                            </div>
                        </div>
                    </div>
                </div>
            </div>
            `;
            
            document.body.insertAdjacentHTML('beforeend', modalHtml);
        },
        
        // 加载任务信息
        loadTaskInfo: async function() {
            try {
                const result = await HttpUtils.get(`/api/v1/tasks/${this.currentTaskId}`);
                
                if (result.code === 200) {
                    const task = result.data;
                    document.getElementById('wizardTaskInfo').innerHTML = `
                        任务名称: <strong>${task.name}</strong> | 
                        同步方向: <strong>${task.source_type} → ${task.target_type}</strong>
                    `;
                    
                    // 保存任务基本信息
                    this.taskData.task = task;
                    
                    // 解析已有配置
                    if (task.config) {
                        try {
                            const config = JSON.parse(task.config);
                            
                            // 填充数据源配置（步骤1）
                            if (config.source_id) {
                                this.taskData.source_id = config.source_id;
                            }
                            if (config.target_id) {
                                this.taskData.target_id = config.target_id;
                            }
                            
                            // 填充表选择配置（步骤2）
                            if (config.selected_databases) {
                                this.taskData.selected_databases = config.selected_databases;
                            }
                            
                            // 填充同步配置（步骤3）
                            if (config.sync_config) {
                                this.taskData.sync_config = config.sync_config;
                            }
                            
                            console.log('已加载现有配置:', this.taskData);
                        } catch (e) {
                            console.warn('配置解析失败，将使用空配置:', e);
                        }
                    }
                    
                    return true;
                } else {
                    console.error('加载任务信息失败:', result.message);
                    return false;
                }
            } catch (error) {
                console.error('加载任务信息失败:', error);
                return false;
            }
        },
        
        // 跳转到指定步骤
        goToStep: function(step) {
            const task = this.taskData.task;
            const isConfigured = task && task.status === 'configured';
            
            // 如果任务已配置，允许跳转到任何步骤
            if (isConfigured) {
                this.showStep(step);
                return;
            }
            
            // 如果任务未配置，只允许跳转到当前步骤或之前的步骤
            if (step <= this.currentStep || step === 1) {
                this.showStep(step);
            } else {
                Toast.warning('请先完成前面的步骤');
            }
        },
        
        // 显示指定步骤
        showStep: function(step) {
            this.currentStep = step;
            
            // 更新步骤指示器
            document.querySelectorAll('.wizard-step').forEach(el => {
                const stepNum = parseInt(el.getAttribute('data-step'));
                el.classList.remove('active', 'completed');
                if (stepNum === step) {
                    el.classList.add('active');
                } else if (stepNum < step) {
                    el.classList.add('completed');
                }
            });
            
            // 更新按钮状态
            document.getElementById('wizardPrevBtn').style.display = step > 1 ? 'inline-block' : 'none';
            document.getElementById('wizardNextBtn').style.display = step < this.totalSteps ? 'inline-block' : 'none';
            document.getElementById('wizardFinishBtn').style.display = step === this.totalSteps ? 'inline-block' : 'none';
            
            // 加载步骤内容
            this.loadStepContent(step);
        },
        
        // 加载步骤内容
        loadStepContent: function(step) {
            const content = document.getElementById('wizardContent');
            
            if (step === 1 && window.TaskWizardStep1) {
                window.TaskWizardStep1.render(content, this.taskData);
            } else if (step === 2 && window.TaskWizardStep2) {
                window.TaskWizardStep2.render(content, this.taskData);
            } else if (step === 3 && window.TaskWizardStep3) {
                window.TaskWizardStep3.render(content, this.taskData);
            } else if (step === 4 && window.TaskWizardStep4) {
                window.TaskWizardStep4.render(content, this.taskData);
            } else {
                content.innerHTML = '<div class="text-center text-muted">步骤内容加载中...</div>';
            }
        },
        
        // 下一步
        nextStep: async function() {
            // 验证当前步骤
            let valid = false;
            
            if (this.currentStep === 1 && window.TaskWizardStep1) {
                valid = await window.TaskWizardStep1.validate(this.taskData);
            } else if (this.currentStep === 2 && window.TaskWizardStep2) {
                valid = await window.TaskWizardStep2.validate(this.taskData);
            } else if (this.currentStep === 3 && window.TaskWizardStep3) {
                valid = await window.TaskWizardStep3.validate(this.taskData);
            }
            
            if (!valid) {
                return;
            }
            
            // 进入下一步
            if (this.currentStep < this.totalSteps) {
                this.showStep(this.currentStep + 1);
            }
        },
        
        // 上一步
        prevStep: function() {
            if (this.currentStep > 1) {
                this.showStep(this.currentStep - 1);
            }
        },
        
        // 完成配置
        finish: async function() {
            try {
                // 提交配置
                const result = await HttpUtils.put(`/api/v1/tasks/${this.currentTaskId}/config`, this.taskData);
                
                if (result.code === 200) {
                    Toast.success('配置保存成功！');
                    this.close();
                    
                    // 刷新任务列表
                    if (window.TaskConfigCore) {
                        window.TaskConfigCore.loadList();
                    }
                } else {
                    Toast.error('配置保存失败: ' + result.message);
                }
            } catch (error) {
                Toast.error('配置保存失败: ' + error.message);
            }
        },
        
        // 关闭向导
        close: function() {
            const modal = document.getElementById('wizardModal');
            if (modal) {
                modal.remove();
            }
        }
    };
})();
