// 任务配置向导 - 主入口文件

// 初始化：创建配置向导模态框HTML
document.addEventListener('DOMContentLoaded', function() {
    const wizardHTML = `
        <!-- 配置向导模态框 -->
        <div id="configWizardModal" style="display:none; position:fixed; top:0; left:0; width:100%; height:100%; background:rgba(0,0,0,0.6); z-index:1001; backdrop-filter: blur(5px);">
            <div style="position:absolute; top:50%; left:50%; transform:translate(-50%,-50%); background:white; padding:0; border-radius:15px; width:65%; max-width:1400px; min-width:800px; height:85vh; max-height:850px; display: flex; flex-direction: column; box-shadow: 0 20px 60px rgba(0,0,0,0.3);">
                <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 20px; border-radius: 15px 15px 0 0; flex-shrink: 0;">
                    <div style="display: flex; justify-content: space-between; align-items: center;">
                        <div style="flex: 1;">
                            <h5 style="color: white; margin: 0;"><i class="bi bi-gear me-2"></i>配置同步任务</h5>
                            <div style="color: rgba(255,255,255,0.9); font-size: 14px; margin-top: 5px;">
                                任务名称: <strong id="wizardTaskName"></strong> | 同步方向: <strong id="wizardSyncDirection"></strong>
                            </div>
                        </div>
                        <div style="display: flex; align-items: flex-start; gap: 8px; margin-left: 30px;">
                            <div class="text-center" style="cursor: pointer;" onclick="jumpToStep(1)">
                                <div id="wizardStepIndicator1" class="wizard-step-indicator active">
                                    <div class="step-number">1</div>
                                    <div class="step-label">选择数据源</div>
                                </div>
                            </div>
                            <div style="color: rgba(255,255,255,0.5); font-size: 18px; margin: 0 5px; padding-top: 7px;">→</div>
                            <div class="text-center" style="cursor: pointer;" onclick="jumpToStep(2)">
                                <div id="wizardStepIndicator2" class="wizard-step-indicator">
                                    <div class="step-number">2</div>
                                    <div class="step-label">选择库和表</div>
                                </div>
                            </div>
                            <div style="color: rgba(255,255,255,0.5); font-size: 18px; margin: 0 5px; padding-top: 7px;">→</div>
                            <div class="text-center" style="cursor: pointer;" onclick="jumpToStep(3)">
                                <div id="wizardStepIndicator3" class="wizard-step-indicator">
                                    <div class="step-number">3</div>
                                    <div class="step-label">配置参数</div>
                                </div>
                            </div>
                            <div style="color: rgba(255,255,255,0.5); font-size: 18px; margin: 0 5px; padding-top: 7px;">→</div>
                            <div class="text-center" style="cursor: pointer;" onclick="jumpToStep(4)">
                                <div id="wizardStepIndicator4" class="wizard-step-indicator">
                                    <div class="step-number">4</div>
                                    <div class="step-label">确认配置</div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                <div style="padding: 30px; flex: 1; overflow-y: auto;">
                    <div id="wizardStep1"><h6 class="mb-3">选择源和目标数据源</h6><div class="row"><div class="col-md-6"><label class="form-label">源数据源 <span class="text-danger">*</span></label><select class="form-select" id="wizardSourceId"><option value="">请选择源数据源</option></select></div><div class="col-md-6"><label class="form-label">目标数据源 <span class="text-danger">*</span></label><select class="form-select" id="wizardTargetId"><option value="">请选择目标数据源</option></select></div></div></div>
                    <div id="wizardStep2" style="display:none;">
                        <h6 class="mb-3">选择要同步的数据库和表</h6>
                        <div class="row">
                            <div class="col-md-5">
                                <div class="card">
                                    <div class="card-header bg-light">
                                        <div class="d-flex justify-content-between align-items-center">
                                            <span><i class="bi bi-list-ul me-2"></i>可选表</span>
                                            <div>
                                                <button type="button" class="btn btn-sm btn-outline-primary me-2" onclick="selectAllAvailable()">
                                                    <i class="bi bi-check-all"></i> 全选
                                                </button>
                                                <span class="badge bg-secondary" id="availableCount">0</span>
                                            </div>
                                        </div>
                                        <div class="input-group input-group-sm mt-2">
                                            <span class="input-group-text"><i class="bi bi-search"></i></span>
                                            <input type="text" class="form-control" id="searchAvailable" placeholder="搜索表名...">
                                        </div>
                                    </div>
                                    <div class="card-body p-0" style="height: 400px; overflow-y: auto;" id="availableList">
                                        <div class="text-center text-muted p-4">
                                            <div class="spinner-border spinner-border-sm me-2" role="status"></div>
                                            加载中...
                                        </div>
                                    </div>
                                </div>
                            </div>
                            <div class="col-md-2 d-flex flex-column justify-content-center align-items-center">
                                <button type="button" class="btn btn-primary mb-3" onclick="moveToSelected()" title="添加选中项">
                                    <i class="bi bi-chevron-right"></i>
                                </button>
                                <button type="button" class="btn btn-primary mb-3" onclick="moveAllToSelected()" title="添加全部">
                                    <i class="bi bi-chevron-double-right"></i>
                                </button>
                                <button type="button" class="btn btn-outline-primary mb-3" onclick="moveToAvailable()" title="移除选中项">
                                    <i class="bi bi-chevron-left"></i>
                                </button>
                                <button type="button" class="btn btn-outline-primary" onclick="moveAllToAvailable()" title="移除全部">
                                    <i class="bi bi-chevron-double-left"></i>
                                </button>
                            </div>
                            <div class="col-md-5">
                                <div class="card">
                                    <div class="card-header bg-light">
                                        <div class="d-flex justify-content-between align-items-center">
                                            <span><i class="bi bi-check-circle me-2"></i>已选表</span>
                                            <div>
                                                <button type="button" class="btn btn-sm btn-outline-success me-2" onclick="showDatabaseRenameModal()" title="批量替换数据库名">
                                                    <i class="bi bi-pencil-square"></i> 替换库名
                                                </button>
                                                <button type="button" class="btn btn-sm btn-outline-info me-2" onclick="showBatchTableRenameModal()" title="批量修改表名">
                                                    <i class="bi bi-pencil"></i> 修改表名
                                                </button>
                                                <button type="button" class="btn btn-sm btn-outline-danger me-2" onclick="clearAllSelected()">
                                                    <i class="bi bi-x-circle"></i> 清空
                                                </button>
                                                <span class="badge bg-primary" id="selectedCount">0</span>
                                            </div>
                                        </div>
                                        <div class="input-group input-group-sm mt-2">
                                            <span class="input-group-text"><i class="bi bi-search"></i></span>
                                            <input type="text" class="form-control" id="searchSelected" placeholder="搜索表名...">
                                        </div>
                                    </div>
                                    <div class="card-body p-0" style="height: 400px; overflow-y: auto;" id="selectedList">
                                        <div class="text-center text-muted p-4">
                                            <i class="bi bi-inbox"></i>
                                            <div class="mt-2">暂无选择</div>
                                        </div>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div id="wizardStep3" style="display:none;">
                        <h6 class="mb-3">配置同步参数</h6>
                        <div class="row">
                            <div class="col-md-6">
                                <div class="mb-3">
                                    <label class="form-label">批次大小</label>
                                    <input type="number" class="form-control" id="wizardBatchSize" min="100" max="10000" value="2500">
                                    <div class="form-text">每批次同步的记录数 (100-10000)</div>
                                </div>
                            </div>
                            <div class="col-md-6">
                                <div class="mb-3">
                                    <label class="form-label">并发线程数</label>
                                    <input type="number" class="form-control" id="wizardThreadCount" min="1" max="20" value="4">
                                    <div class="form-text">并发执行的线程数 (1-20)</div>
                                </div>
                            </div>
                        </div>
                        <div class="row">
                            <div class="col-md-6">
                                <div class="mb-3">
                                    <label class="form-label d-block">同步模式</label>
                                    <div class="form-check">
                                        <input class="form-check-input" type="radio" name="wizardSyncMode" id="syncModeFull" value="full" checked>
                                        <label class="form-check-label" for="syncModeFull">全量同步</label>
                                        <div class="form-text">只同步一次全量数据，完成后不再同步</div>
                                    </div>
                                    <div class="form-check">
                                        <input class="form-check-input" type="radio" name="wizardSyncMode" id="syncModeIncremental" value="incremental">
                                        <label class="form-check-label" for="syncModeIncremental">增量同步</label>
                                        <div class="form-text">全量同步后监听 binlog 持续同步增量数据</div>
                                    </div>
                                </div>
                            </div>
                            <div class="col-md-6">
                                <div class="mb-3">
                                    <label class="form-label d-block">错误处理策略</label>
                                    <div class="form-check form-check-inline">
                                        <input class="form-check-input" type="radio" name="wizardErrorStrategy" id="errorStrategySkip" value="skip" checked>
                                        <label class="form-check-label" for="errorStrategySkip">跳过错误继续</label>
                                    </div>
                                    <div class="form-check form-check-inline">
                                        <input class="form-check-input" type="radio" name="wizardErrorStrategy" id="errorStrategyPause" value="pause">
                                        <label class="form-check-label" for="errorStrategyPause">遇错暂停</label>
                                    </div>
                                </div>
                            </div>
                        </div>
                        <div class="row">
                            <div class="col-md-6">
                                <div class="mb-3">
                                    <label class="form-label d-block">目标表存在策略</label>
                                    <div class="form-check">
                                        <input class="form-check-input" type="radio" name="wizardTableExistsStrategy" id="tableStrategyDrop" value="drop" checked>
                                        <label class="form-check-label" for="tableStrategyDrop">删除重建</label>
                                    </div>
                                    <div class="form-check">
                                        <input class="form-check-input" type="radio" name="wizardTableExistsStrategy" id="tableStrategyTruncate" value="truncate">
                                        <label class="form-check-label" for="tableStrategyTruncate">清空数据</label>
                                    </div>
                                    <div class="form-check">
                                        <input class="form-check-input" type="radio" name="wizardTableExistsStrategy" id="tableStrategyBackup" value="backup">
                                        <label class="form-check-label" for="tableStrategyBackup">备份后重建</label>
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                    <div id="wizardStep4" style="display:none; height: 100%;">
                        <div class="alert alert-info">
                            <i class="bi bi-info-circle me-2"></i>请仔细检查以下配置，确认无误后点击"保存配置"
                        </div>
                        
                        <div class="row" style="height: calc(100% - 70px);">
                            <!-- 左侧：配置信息 -->
                            <div class="col-md-6" style="height: 100%;">
                                <div class="card" style="height: 100%; display: flex; flex-direction: column;">
                                    <div class="card-header bg-light" style="flex-shrink: 0;">
                                        <strong><i class="bi bi-gear me-2"></i>配置信息</strong>
                                    </div>
                                    <div class="card-body" style="flex: 1; overflow-y: auto; padding: 20px;">
                                        <div class="row mb-3">
                                            <div class="col-5" style="color: #718096; font-size: 14px;">源数据源</div>
                                            <div class="col-7" style="color: #2d3748; font-size: 14px; font-weight: 500;" id="summarySourceId"></div>
                                        </div>
                                        <div class="row mb-3">
                                            <div class="col-5" style="color: #718096; font-size: 14px;">目标数据源</div>
                                            <div class="col-7" style="color: #2d3748; font-size: 14px; font-weight: 500;" id="summaryTargetId"></div>
                                        </div>
                                        <div class="row mb-3">
                                            <div class="col-5" style="color: #718096; font-size: 14px;">同步表数量</div>
                                            <div class="col-7" style="color: #667eea; font-size: 14px; font-weight: 600;" id="summaryTableCount"></div>
                                        </div>
                                        <div class="row mb-3">
                                            <div class="col-5" style="color: #718096; font-size: 14px;">同步模式</div>
                                            <div class="col-7" style="color: #2d3748; font-size: 14px; font-weight: 500;" id="summarySyncMode"></div>
                                        </div>
                                        <div class="row mb-3">
                                            <div class="col-5" style="color: #718096; font-size: 14px;">批次大小</div>
                                            <div class="col-7" style="color: #2d3748; font-size: 14px;"><span id="summaryBatchSize" style="font-weight: 600; color: #667eea;"></span> <span style="color: #718096;">条/批次</span></div>
                                        </div>
                                        <div class="row mb-3">
                                            <div class="col-5" style="color: #718096; font-size: 14px;">并发线程数</div>
                                            <div class="col-7" style="color: #2d3748; font-size: 14px;"><span id="summaryThreadCount" style="font-weight: 600; color: #667eea;"></span> <span style="color: #718096;">个线程</span></div>
                                        </div>
                                        <div class="row mb-3">
                                            <div class="col-5" style="color: #718096; font-size: 14px;">错误处理策略</div>
                                            <div class="col-7" style="color: #2d3748; font-size: 14px; font-weight: 500;" id="summaryErrorStrategy"></div>
                                        </div>
                                        <div class="row mb-0">
                                            <div class="col-5" style="color: #718096; font-size: 14px;">目标表存在策略</div>
                                            <div class="col-7" style="color: #2d3748; font-size: 14px; font-weight: 500;" id="summaryTableExistsStrategy"></div>
                                        </div>
                                    </div>
                                </div>
                            </div>
                            
                            <!-- 右侧：选中的表 -->
                            <div class="col-md-6" style="height: 100%;">
                                <div class="card" style="height: 100%; display: flex; flex-direction: column;">
                                    <div class="card-header bg-light" style="flex-shrink: 0;">
                                        <strong><i class="bi bi-table me-2"></i>选中的表</strong>
                                    </div>
                                    <div class="card-body p-2" style="flex: 1; overflow-y: auto;" id="summaryTableList">
                                        <!-- 动态加载 -->
                                    </div>
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                <div style="padding: 20px; border-top: 1px solid #e2e8f0; background: #f7fafc; border-radius: 0 0 15px 15px; flex-shrink: 0;">
                    <div class="d-flex justify-content-between">
                        <button type="button" class="btn btn-secondary" onclick="closeConfigWizard()">取消</button>
                        <div>
                            <button type="button" class="btn btn-outline-secondary me-2" id="wizardPrevBtn" onclick="wizardPrevStep()">
                                <i class="bi bi-arrow-left"></i> 上一步
                            </button>
                            <button type="button" class="btn btn-primary" id="wizardNextBtn" onclick="wizardNextStep()">
                                下一步 <i class="bi bi-arrow-right"></i>
                            </button>
                            <button type="button" class="btn btn-success" id="wizardSaveBtn" style="display:none;" onclick="saveWizardConfig()">
                                <i class="bi bi-check-circle"></i> 保存配置
                            </button>
                        </div>
                    </div>
                </div>
            </div>
        </div>
        
        <!-- 确认对话框 -->
        <div id="confirmModal" style="display:none; position:fixed; top:0; left:0; width:100%; height:100%; background:rgba(0,0,0,0.6); z-index:2000; backdrop-filter: blur(5px);">
            <div style="position:absolute; top:50%; left:50%; transform:translate(-50%,-50%); background:white; padding:0; border-radius:15px; width:450px; box-shadow: 0 20px 60px rgba(0,0,0,0.3);">
                <div style="background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%); padding: 20px; border-radius: 15px 15px 0 0;">
                    <h5 style="color: white; margin: 0;"><i class="bi bi-exclamation-triangle me-2"></i>确认操作</h5>
                </div>
                <div style="padding: 30px;">
                    <p id="confirmMessage" style="font-size: 16px; color: #2d3748; margin: 0;"></p>
                    <p style="font-size: 14px; color: #718096; margin-top: 10px; margin-bottom: 0;">此操作不可恢复，请谨慎操作。</p>
                </div>
                <div style="padding: 20px; border-top: 1px solid #e2e8f0; background: #f7fafc; border-radius: 0 0 15px 15px; display: flex; justify-content: flex-end; gap: 10px;">
                    <button type="button" class="btn btn-secondary" onclick="closeConfirmModal(false)">取消</button>
                    <button type="button" class="btn btn-danger" onclick="closeConfirmModal(true)"><i class="bi bi-check-circle me-1"></i>确定</button>
                </div>
            </div>
        </div>
        
        <!-- 数据库名替换模态框 -->
        <div id="databaseRenameModal" style="display:none; position:fixed; top:0; left:0; width:100%; height:100%; background:rgba(0,0,0,0.6); z-index:2001; backdrop-filter: blur(5px);">
            <div style="position:absolute; top:50%; left:50%; transform:translate(-50%,-50%); background:white; padding:0; border-radius:15px; width:500px; box-shadow: 0 20px 60px rgba(0,0,0,0.3);">
                <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 20px; border-radius: 15px 15px 0 0;">
                    <h5 style="color: white; margin: 0;"><i class="bi bi-pencil-square me-2"></i>批量替换数据库名</h5>
                </div>
                <div style="padding: 30px;">
                    <div class="mb-3">
                        <label class="form-label">替换类型</label>
                        <div class="form-check">
                            <input class="form-check-input" type="radio" name="renameType" id="renameTypePrefix" value="prefix" checked>
                            <label class="form-check-label" for="renameTypePrefix">替换前缀</label>
                        </div>
                        <div class="form-check">
                            <input class="form-check-input" type="radio" name="renameType" id="renameTypeSuffix" value="suffix">
                            <label class="form-check-label" for="renameTypeSuffix">替换后缀</label>
                        </div>
                    </div>
                    <div class="mb-3">
                        <label class="form-label">原文本</label>
                        <input type="text" class="form-control" id="renameOldText" placeholder="例如: test_">
                        <div class="form-text">要被替换的前缀或后缀</div>
                    </div>
                    <div class="mb-3">
                        <label class="form-label">新文本</label>
                        <input type="text" class="form-control" id="renameNewText" placeholder="例如: prod_">
                        <div class="form-text">替换后的前缀或后缀（可为空）</div>
                    </div>
                    <div class="alert alert-info mb-0">
                        <i class="bi bi-info-circle me-2"></i>
                        <strong>示例：</strong><br>
                        前缀替换：test_app → prod_app<br>
                        后缀替换：app_test → app_prod
                    </div>
                </div>
                <div style="padding: 20px; border-top: 1px solid #e2e8f0; background: #f7fafc; border-radius: 0 0 15px 15px; display: flex; justify-content: flex-end; gap: 10px;">
                    <button type="button" class="btn btn-secondary" onclick="closeDatabaseRenameModal()">取消</button>
                    <button type="button" class="btn btn-primary" onclick="applyDatabaseRename()"><i class="bi bi-check-circle me-1"></i>应用</button>
                </div>
            </div>
        </div>
        
        <!-- 单个表名修改模态框 -->
        <div id="tableRenameModal" style="display:none; position:fixed; top:0; left:0; width:100%; height:100%; background:rgba(0,0,0,0.6); z-index:2002; backdrop-filter: blur(5px);">
            <div style="position:absolute; top:50%; left:50%; transform:translate(-50%,-50%); background:white; padding:0; border-radius:15px; width:500px; box-shadow: 0 20px 60px rgba(0,0,0,0.3);">
                <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 20px; border-radius: 15px 15px 0 0;">
                    <h5 style="color: white; margin: 0;"><i class="bi bi-pencil me-2"></i>修改表名</h5>
                </div>
                <div style="padding: 30px;">
                    <div class="mb-3">
                        <label class="form-label">源表名</label>
                        <div class="form-control bg-light" id="tableRenameSourceName" style="background: #f7fafc;"></div>
                    </div>
                    <div class="mb-3">
                        <label class="form-label">目标表名</label>
                        <input type="text" class="form-control" id="tableRenameTargetName" placeholder="输入新的表名">
                        <div class="form-text">修改后的目标表名</div>
                    </div>
                </div>
                <div style="padding: 20px; border-top: 1px solid #e2e8f0; background: #f7fafc; border-radius: 0 0 15px 15px; display: flex; justify-content: flex-end; gap: 10px;">
                    <button type="button" class="btn btn-secondary" onclick="closeTableRenameModal()">取消</button>
                    <button type="button" class="btn btn-primary" onclick="applyTableRename()"><i class="bi bi-check-circle me-1"></i>确定</button>
                </div>
            </div>
        </div>
        
        <!-- 批量修改表名模态框 -->
        <div id="batchTableRenameModal" style="display:none; position:fixed; top:0; left:0; width:100%; height:100%; background:rgba(0,0,0,0.6); z-index:2003; backdrop-filter: blur(5px);">
            <div style="position:absolute; top:50%; left:50%; transform:translate(-50%,-50%); background:white; padding:0; border-radius:15px; width:550px; box-shadow: 0 20px 60px rgba(0,0,0,0.3);">
                <div style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%); padding: 20px; border-radius: 15px 15px 0 0;">
                    <h5 style="color: white; margin: 0;"><i class="bi bi-pencil-square me-2"></i>批量修改表名</h5>
                </div>
                <div style="padding: 30px;">
                    <div class="mb-3">
                        <label class="form-label">操作类型</label>
                        <div class="form-check">
                            <input class="form-check-input" type="radio" name="batchTableRenameType" id="batchTableRenameAddPrefix" value="add_prefix" checked>
                            <label class="form-check-label" for="batchTableRenameAddPrefix">添加前缀</label>
                        </div>
                        <div class="form-check">
                            <input class="form-check-input" type="radio" name="batchTableRenameType" id="batchTableRenameAddSuffix" value="add_suffix">
                            <label class="form-check-label" for="batchTableRenameAddSuffix">添加后缀</label>
                        </div>
                        <div class="form-check">
                            <input class="form-check-input" type="radio" name="batchTableRenameType" id="batchTableRenameReplacePrefix" value="replace_prefix">
                            <label class="form-check-label" for="batchTableRenameReplacePrefix">替换前缀</label>
                        </div>
                        <div class="form-check">
                            <input class="form-check-input" type="radio" name="batchTableRenameType" id="batchTableRenameReplaceSuffix" value="replace_suffix">
                            <label class="form-check-label" for="batchTableRenameReplaceSuffix">替换后缀</label>
                        </div>
                    </div>
                    <div class="mb-3" id="batchTableRenameOldTextGroup" style="display: none;">
                        <label class="form-label">原文本</label>
                        <input type="text" class="form-control" id="batchTableRenameOldText" placeholder="要被替换的前缀或后缀">
                    </div>
                    <div class="mb-3">
                        <label class="form-label">新文本</label>
                        <input type="text" class="form-control" id="batchTableRenameNewText" placeholder="新的前缀或后缀">
                    </div>
                    <div class="alert alert-info mb-0">
                        <i class="bi bi-info-circle me-2"></i>
                        <strong>示例：</strong><br>
                        添加前缀：orders → bak_orders<br>
                        添加后缀：orders → orders_bak<br>
                        替换前缀：test_orders → prod_orders<br>
                        替换后缀：orders_test → orders_prod
                    </div>
                </div>
                <div style="padding: 20px; border-top: 1px solid #e2e8f0; background: #f7fafc; border-radius: 0 0 15px 15px; display: flex; justify-content: flex-end; gap: 10px;">
                    <button type="button" class="btn btn-secondary" onclick="closeBatchTableRenameModal()">取消</button>
                    <button type="button" class="btn btn-primary" onclick="applyBatchTableRename()"><i class="bi bi-check-circle me-1"></i>应用</button>
                </div>
            </div>
        </div>
        
        <style>
        .wizard-step-indicator .step-number { width: 32px; height: 32px; border-radius: 50%; background: rgba(255, 255, 255, 0.2); color: rgba(255, 255, 255, 0.7); display: flex; align-items: center; justify-content: center; font-weight: 600; margin: 0 auto 6px; transition: all 0.3s; border: 2px solid rgba(255, 255, 255, 0.3); font-size: 14px; }
        .wizard-step-indicator.active .step-number { background: white; color: #667eea; box-shadow: 0 4px 15px rgba(255, 255, 255, 0.3); border-color: white; }
        .wizard-step-indicator.completed .step-number { background: rgba(72, 187, 120, 0.9); color: white; border-color: #48bb78; }
        .wizard-step-indicator.disabled { opacity: 0.4; cursor: not-allowed !important; }
        .wizard-step-indicator:not(.disabled):hover .step-number { transform: scale(1.1); box-shadow: 0 4px 12px rgba(255, 255, 255, 0.2); }
        .wizard-step-indicator .step-label { font-size: 11px; color: rgba(255, 255, 255, 0.8); white-space: nowrap; }
        .wizard-step-indicator.active .step-label { color: white; font-weight: 600; }
        .wizard-step-indicator.completed .step-label { color: rgba(72, 187, 120, 1); font-weight: 600; }
        .transfer-group { border-bottom: 1px solid #e9ecef; }
        .transfer-group:last-child { border-bottom: none; }
        .transfer-group-header { background: #f8f9fa; padding: 10px 15px; border-bottom: 1px solid #e9ecef; }
        .transfer-item { padding: 8px 15px; border-bottom: 1px solid #f8f9fa; transition: background 0.2s; }
        .transfer-item:hover { background: #f8f9fa; }
        .transfer-item:last-child { border-bottom: none; }
        .card-header { position: sticky; top: 0; z-index: 1; }
        </style>
    `;
    
    document.body.insertAdjacentHTML('beforeend', wizardHTML);
});


// 监听批量修改表名类型变化，动态显示/隐藏"原文本"输入框
document.addEventListener('DOMContentLoaded', function() {
    const radioButtons = document.getElementsByName('batchTableRenameType');
    const oldTextGroup = document.getElementById('batchTableRenameOldTextGroup');
    
    if (radioButtons && oldTextGroup) {
        radioButtons.forEach(radio => {
            radio.addEventListener('change', function() {
                const value = this.value;
                if (value === 'replace_prefix' || value === 'replace_suffix') {
                    oldTextGroup.style.display = 'block';
                } else {
                    oldTextGroup.style.display = 'none';
                }
            });
        });
    }
});


// 全局 ESC 键监听 - 关闭所有模态框
document.addEventListener('keydown', function(e) {
    if (e.key === 'Escape' || e.keyCode === 27) {
        // 按优先级关闭模态框（从最上层到最底层）
        
        // 1. 批量修改表名模态框
        const batchTableRenameModal = document.getElementById('batchTableRenameModal');
        if (batchTableRenameModal && batchTableRenameModal.style.display !== 'none') {
            closeBatchTableRenameModal();
            return;
        }
        
        // 2. 单个表名修改模态框
        const tableRenameModal = document.getElementById('tableRenameModal');
        if (tableRenameModal && tableRenameModal.style.display !== 'none') {
            closeTableRenameModal();
            return;
        }
        
        // 3. 数据库名替换模态框
        const databaseRenameModal = document.getElementById('databaseRenameModal');
        if (databaseRenameModal && databaseRenameModal.style.display !== 'none') {
            closeDatabaseRenameModal();
            return;
        }
        
        // 4. 确认对话框
        const confirmModal = document.getElementById('confirmModal');
        if (confirmModal && confirmModal.style.display !== 'none') {
            closeConfirmModal(false);
            return;
        }
        
        // 5. 配置向导模态框
        const configWizardModal = document.getElementById('configWizardModal');
        if (configWizardModal && configWizardModal.style.display !== 'none') {
            closeConfigWizard();
            return;
        }
    }
});
