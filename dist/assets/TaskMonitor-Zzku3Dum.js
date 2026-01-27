import{d as H,r as $,ao as we,h as x,g as N,ap as ee,ac as _e,C as R,be as Ke,y as L,A as D,z as le,U as Ze,bj as Qe,I as et,q as Pe,aH as Se,aG as tt,G as at,J as Le,u as ve,m as Ie,K as Me,L as Be,ae as st,T as nt,H as ie,o as p,c as w,a as f,bk as lt,ba as rt,e as d,w as i,f as a,bl as ot,bm as it,bn as ut,N as U,F as J,j as K,k as B,l as E,bo as ct,bp as Ce,at as Te,t as b,i as O,au as se}from"./index-DVUYEo1S.js";import{u as dt,S as $e,A as ft,N as mt,a as vt}from"./syncTask-CaCuPa9_.js";import{t as gt,l as pt,v as ht,w as yt,x as X,u as bt,h as ue,B as ne,R as xt,k as ze,b as ce,a as Z,e as ye,c as be,_ as _t}from"./_plugin-vue_export-helper-D0l3HQ5W.js";import{l as me,N as xe,a as wt,b as de}from"./event-BUlYtK1V.js";const Ne=H({name:"SlotMachineNumber",props:{clsPrefix:{type:String,required:!0},value:{type:[Number,String],required:!0},oldOriginalNumber:{type:Number,default:void 0},newOriginalNumber:{type:Number,default:void 0}},setup(t){const l=$(null),s=$(t.value),g=$(t.value),m=$("up"),n=$(!1),v=N(()=>n.value?`${t.clsPrefix}-base-slot-machine-current-number--${m.value}-scroll`:null),k=N(()=>n.value?`${t.clsPrefix}-base-slot-machine-old-number--${m.value}-scroll`:null);we(_e(t,"value"),(h,S)=>{s.value=S,g.value=h,ee(I)});function I(){const h=t.newOriginalNumber,S=t.oldOriginalNumber;S===void 0||h===void 0||(h>S?M("up"):S>h&&M("down"))}function M(h){m.value=h,n.value=!1,ee(()=>{var S;(S=l.value)===null||S===void 0||S.offsetWidth,n.value=!0})}return()=>{const{clsPrefix:h}=t;return x("span",{ref:l,class:`${h}-base-slot-machine-number`},s.value!==null?x("span",{class:[`${h}-base-slot-machine-old-number ${h}-base-slot-machine-old-number--top`,k.value]},s.value):null,x("span",{class:[`${h}-base-slot-machine-current-number`,v.value]},x("span",{ref:"numberWrapper",class:[`${h}-base-slot-machine-current-number__inner`,typeof t.value!="number"&&`${h}-base-slot-machine-current-number__inner--not-number`]},g.value)),s.value!==null?x("span",{class:[`${h}-base-slot-machine-old-number ${h}-base-slot-machine-old-number--bottom`,k.value]},s.value):null)}}}),{cubicBezierEaseOut:Q}=Ke;function kt({duration:t=".2s"}={}){return[R("&.fade-up-width-expand-transition-leave-active",{transition:`
 opacity ${t} ${Q},
 max-width ${t} ${Q},
 transform ${t} ${Q}
 `}),R("&.fade-up-width-expand-transition-enter-active",{transition:`
 opacity ${t} ${Q},
 max-width ${t} ${Q},
 transform ${t} ${Q}
 `}),R("&.fade-up-width-expand-transition-enter-to",{opacity:1,transform:"translateX(0) translateY(0)"}),R("&.fade-up-width-expand-transition-enter-from",{maxWidth:"0 !important",opacity:0,transform:"translateY(60%)"}),R("&.fade-up-width-expand-transition-leave-from",{opacity:1,transform:"translateY(0)"}),R("&.fade-up-width-expand-transition-leave-to",{maxWidth:"0 !important",opacity:0,transform:"translateY(60%)"})]}const St=R([R("@keyframes n-base-slot-machine-fade-up-in",`
 from {
 transform: translateY(60%);
 opacity: 0;
 }
 to {
 transform: translateY(0);
 opacity: 1;
 }
 `),R("@keyframes n-base-slot-machine-fade-down-in",`
 from {
 transform: translateY(-60%);
 opacity: 0;
 }
 to {
 transform: translateY(0);
 opacity: 1;
 }
 `),R("@keyframes n-base-slot-machine-fade-up-out",`
 from {
 transform: translateY(0%);
 opacity: 1;
 }
 to {
 transform: translateY(-60%);
 opacity: 0;
 }
 `),R("@keyframes n-base-slot-machine-fade-down-out",`
 from {
 transform: translateY(0%);
 opacity: 1;
 }
 to {
 transform: translateY(60%);
 opacity: 0;
 }
 `),L("base-slot-machine",`
 overflow: hidden;
 white-space: nowrap;
 display: inline-block;
 height: 18px;
 line-height: 18px;
 `,[L("base-slot-machine-number",`
 display: inline-block;
 position: relative;
 height: 18px;
 width: .6em;
 max-width: .6em;
 `,[kt({duration:".2s"}),gt({duration:".2s",delay:"0s"}),L("base-slot-machine-old-number",`
 display: inline-block;
 opacity: 0;
 position: absolute;
 left: 0;
 right: 0;
 `,[D("top",{transform:"translateY(-100%)"}),D("bottom",{transform:"translateY(100%)"}),D("down-scroll",{animation:"n-base-slot-machine-fade-down-out .2s cubic-bezier(0, 0, .2, 1)",animationIterationCount:1}),D("up-scroll",{animation:"n-base-slot-machine-fade-up-out .2s cubic-bezier(0, 0, .2, 1)",animationIterationCount:1})]),L("base-slot-machine-current-number",`
 display: inline-block;
 position: absolute;
 left: 0;
 top: 0;
 bottom: 0;
 right: 0;
 opacity: 1;
 transform: translateY(0);
 width: .6em;
 `,[D("down-scroll",{animation:"n-base-slot-machine-fade-down-in .2s cubic-bezier(0, 0, .2, 1)",animationIterationCount:1}),D("up-scroll",{animation:"n-base-slot-machine-fade-up-in .2s cubic-bezier(0, 0, .2, 1)",animationIterationCount:1}),le("inner",`
 display: inline-block;
 position: absolute;
 right: 0;
 top: 0;
 width: .6em;
 `,[D("not-number",`
 right: unset;
 left: 0;
 `)])])])])]),Ct=H({name:"BaseSlotMachine",props:{clsPrefix:{type:String,required:!0},value:{type:[Number,String],default:0},max:{type:Number,default:void 0},appeared:{type:Boolean,required:!0}},setup(t){Ze("-base-slot-machine",St,_e(t,"clsPrefix"));const l=$(),s=$(),g=N(()=>{if(typeof t.value=="string")return[];if(t.value<1)return[0];const m=[];let n=t.value;for(t.max!==void 0&&(n=Math.min(t.max,n));n>=1;)m.push(n%10),n/=10,n=Math.floor(n);return m.reverse(),m});return we(_e(t,"value"),(m,n)=>{typeof m=="string"?(s.value=void 0,l.value=void 0):typeof n=="string"?(s.value=m,l.value=void 0):(s.value=m,l.value=n)}),()=>{const{value:m,clsPrefix:n}=t;return typeof m=="number"?x("span",{class:`${n}-base-slot-machine`},x(Qe,{name:"fade-up-width-expand-transition",tag:"span"},{default:()=>g.value.map((v,k)=>x(Ne,{clsPrefix:n,key:g.value.length-k-1,oldOriginalNumber:l.value,newOriginalNumber:s.value,value:v}))}),x(et,{key:"+",width:!0},{default:()=>t.max!==void 0&&t.max<m?x(Ne,{clsPrefix:n,value:"+"}):null})):x("span",{class:`${n}-base-slot-machine`},m)}}});function Tt(t){const{errorColor:l,infoColor:s,successColor:g,warningColor:m,fontFamily:n}=t;return{color:l,colorInfo:s,colorSuccess:g,colorError:l,colorWarning:m,fontSize:"12px",fontFamily:n}}const $t={common:Pe,self:Tt},zt=R([R("@keyframes badge-wave-spread",{from:{boxShadow:"0 0 0.5px 0px var(--n-ripple-color)",opacity:.6},to:{boxShadow:"0 0 0.5px 4.5px var(--n-ripple-color)",opacity:0}}),L("badge",`
 display: inline-flex;
 position: relative;
 vertical-align: middle;
 font-family: var(--n-font-family);
 `,[D("as-is",[L("badge-sup",{position:"static",transform:"translateX(0)"},[Se({transformOrigin:"left bottom",originalTransform:"translateX(0)"})])]),D("dot",[L("badge-sup",`
 height: 8px;
 width: 8px;
 padding: 0;
 min-width: 8px;
 left: 100%;
 bottom: calc(100% - 4px);
 `,[R("::before","border-radius: 4px;")])]),L("badge-sup",`
 background: var(--n-color);
 transition:
 background-color .3s var(--n-bezier),
 color .3s var(--n-bezier);
 color: #FFF;
 position: absolute;
 height: 18px;
 line-height: 18px;
 border-radius: 9px;
 padding: 0 6px;
 text-align: center;
 font-size: var(--n-font-size);
 transform: translateX(-50%);
 left: 100%;
 bottom: calc(100% - 9px);
 font-variant-numeric: tabular-nums;
 z-index: 2;
 display: flex;
 align-items: center;
 `,[Se({transformOrigin:"left bottom",originalTransform:"translateX(-50%)"}),L("base-wave",{zIndex:1,animationDuration:"2s",animationIterationCount:"infinite",animationDelay:"1s",animationTimingFunction:"var(--n-ripple-bezier)",animationName:"badge-wave-spread"}),R("&::before",`
 opacity: 0;
 transform: scale(1);
 border-radius: 9px;
 content: "";
 position: absolute;
 left: 0;
 right: 0;
 top: 0;
 bottom: 0;
 `)])])]),Nt=Object.assign(Object.assign({},ve.props),{value:[String,Number],max:Number,dot:Boolean,type:{type:String,default:"default"},show:{type:Boolean,default:!0},showZero:Boolean,processing:Boolean,color:String,offset:Array}),fe=H({name:"Badge",props:Nt,setup(t,{slots:l}){const{mergedClsPrefixRef:s,inlineThemeDisabled:g,mergedRtlRef:m}=Le(t),n=ve("Badge","-badge",zt,$t,t,s),v=$(!1),k=()=>{v.value=!0},I=()=>{v.value=!1},M=N(()=>t.show&&(t.dot||t.value!==void 0&&!(!t.showZero&&Number(t.value)<=0)||!st(l.value)));Ie(()=>{M.value&&(v.value=!0)});const h=Me("Badge",m,s),S=N(()=>{const{type:z,color:C}=t,{common:{cubicBezierEaseInOut:y,cubicBezierEaseOut:Y},self:{[nt("color",z)]:F,fontFamily:te,fontSize:c}}=n.value;return{"--n-font-size":c,"--n-font-family":te,"--n-color":C||F,"--n-ripple-color":C||F,"--n-bezier":y,"--n-ripple-bezier":Y}}),P=g?Be("badge",N(()=>{let z="";const{type:C,color:y}=t;return C&&(z+=C[0]),y&&(z+=yt(y)),z}),S,t):void 0,V=N(()=>{const{offset:z}=t;if(!z)return;const[C,y]=z,Y=typeof C=="number"?`${C}px`:C,F=typeof y=="number"?`${y}px`:y;return{transform:`translate(calc(${h!=null&&h.value?"50%":"-50%"} + ${Y}), ${F})`}});return{rtlEnabled:h,mergedClsPrefix:s,appeared:v,showBadge:M,handleAfterEnter:k,handleAfterLeave:I,cssVars:g?void 0:S,themeClass:P==null?void 0:P.themeClass,onRender:P==null?void 0:P.onRender,offsetStyle:V}},render(){var t;const{mergedClsPrefix:l,onRender:s,themeClass:g,$slots:m}=this;s==null||s();const n=(t=m.default)===null||t===void 0?void 0:t.call(m);return x("div",{class:[`${l}-badge`,this.rtlEnabled&&`${l}-badge--rtl`,g,{[`${l}-badge--dot`]:this.dot,[`${l}-badge--as-is`]:!n}],style:this.cssVars},n,x(tt,{name:"fade-in-scale-up-transition",onAfterEnter:this.handleAfterEnter,onAfterLeave:this.handleAfterLeave},{default:()=>this.showBadge?x("sup",{class:`${l}-badge-sup`,title:pt(this.value),style:this.offsetStyle},at(m.value,()=>[this.dot?null:x(Ct,{clsPrefix:l,appeared:this.appeared,max:this.max,value:this.value})]),this.processing?x(ht,{clsPrefix:l}):null):null}))}});function Rt(t){const{textColor2:l,textColor3:s,fontSize:g,fontWeight:m}=t;return{labelFontSize:g,labelFontWeight:m,valueFontWeight:m,valueFontSize:"24px",labelTextColor:s,valuePrefixTextColor:l,valueSuffixTextColor:l,valueTextColor:l}}const Et={common:Pe,self:Rt},Pt=L("statistic",[le("label",`
 font-weight: var(--n-label-font-weight);
 transition: .3s color var(--n-bezier);
 font-size: var(--n-label-font-size);
 color: var(--n-label-text-color);
 `),L("statistic-value",`
 margin-top: 4px;
 font-weight: var(--n-value-font-weight);
 `,[le("prefix",`
 margin: 0 4px 0 0;
 font-size: var(--n-value-font-size);
 transition: .3s color var(--n-bezier);
 color: var(--n-value-prefix-text-color);
 `,[L("icon",{verticalAlign:"-0.125em"})]),le("content",`
 font-size: var(--n-value-font-size);
 transition: .3s color var(--n-bezier);
 color: var(--n-value-text-color);
 `),le("suffix",`
 margin: 0 0 0 4px;
 font-size: var(--n-value-font-size);
 transition: .3s color var(--n-bezier);
 color: var(--n-value-suffix-text-color);
 `,[L("icon",{verticalAlign:"-0.125em"})])])]),Lt=Object.assign(Object.assign({},ve.props),{tabularNums:Boolean,label:String,value:[String,Number]}),Re=H({name:"Statistic",props:Lt,slots:Object,setup(t){const{mergedClsPrefixRef:l,inlineThemeDisabled:s,mergedRtlRef:g}=Le(t),m=ve("Statistic","-statistic",Pt,Et,t,l),n=Me("Statistic",g,l),v=N(()=>{const{self:{labelFontWeight:I,valueFontSize:M,valueFontWeight:h,valuePrefixTextColor:S,labelTextColor:P,valueSuffixTextColor:V,valueTextColor:z,labelFontSize:C},common:{cubicBezierEaseInOut:y}}=m.value;return{"--n-bezier":y,"--n-label-font-size":C,"--n-label-font-weight":I,"--n-label-text-color":P,"--n-value-font-weight":h,"--n-value-font-size":M,"--n-value-prefix-text-color":S,"--n-value-suffix-text-color":V,"--n-value-text-color":z}}),k=s?Be("statistic",void 0,v,t):void 0;return{rtlEnabled:n,mergedClsPrefix:l,cssVars:s?void 0:v,themeClass:k==null?void 0:k.themeClass,onRender:k==null?void 0:k.onRender}},render(){var t;const{mergedClsPrefix:l,$slots:{default:s,label:g,prefix:m,suffix:n}}=this;return(t=this.onRender)===null||t===void 0||t.call(this),x("div",{class:[`${l}-statistic`,this.themeClass,this.rtlEnabled&&`${l}-statistic--rtl`],style:this.cssVars},ie(g,v=>x("div",{class:`${l}-statistic__label`},this.label||v)),x("div",{class:`${l}-statistic-value`,style:{fontVariantNumeric:this.tabularNums?"tabular-nums":""}},ie(m,v=>v&&x("span",{class:`${l}-statistic-value__prefix`},v)),this.value!==void 0?x("span",{class:`${l}-statistic-value__content`},this.value):ie(s,v=>v&&x("span",{class:`${l}-statistic-value__content`},v)),ie(n,v=>v&&x("span",{class:`${l}-statistic-value__suffix`},v))))}}),It={xmlns:"http://www.w3.org/2000/svg","xmlns:xlink":"http://www.w3.org/1999/xlink",viewBox:"0 0 512 512"},Mt=H({name:"Pause",render:function(l,s){return p(),w("svg",It,s[0]||(s[0]=[f("path",{d:"M208 432h-48a16 16 0 0 1-16-16V96a16 16 0 0 1 16-16h48a16 16 0 0 1 16 16v320a16 16 0 0 1-16 16z",fill:"currentColor"},null,-1),f("path",{d:"M352 432h-48a16 16 0 0 1-16-16V96a16 16 0 0 1 16-16h48a16 16 0 0 1 16 16v320a16 16 0 0 1-16 16z",fill:"currentColor"},null,-1)]))}}),Bt={xmlns:"http://www.w3.org/2000/svg","xmlns:xlink":"http://www.w3.org/1999/xlink",viewBox:"0 0 512 512"},Ee=H({name:"Play",render:function(l,s){return p(),w("svg",Bt,s[0]||(s[0]=[f("path",{d:"M133 440a35.37 35.37 0 0 1-17.5-4.67c-12-6.8-19.46-20-19.46-34.33V111c0-14.37 7.46-27.53 19.46-34.33a35.13 35.13 0 0 1 35.77.45l247.85 148.36a36 36 0 0 1 0 61l-247.89 148.4A35.5 35.5 0 0 1 133 440z",fill:"currentColor"},null,-1)]))}}),Ot=lt("taskMonitor",()=>{const t=$(null),l=$([]),s=$(!1),g=$(null);let m=null,n=null;const v=N(()=>{var c;return((c=t.value)==null?void 0:c.status)==="running"}),k=N(()=>{var c;return((c=t.value)==null?void 0:c.status)==="paused"}),I=N(()=>l.value.length);async function M(c){s.value=!0,g.value=null;try{t.value=null,l.value=[],await C();const o=await X("get_task",{id:c});if(!o)throw new Error("任务不存在");let G={mysqlConfig:void 0,esConfig:void 0,syncConfig:{}};try{o.config&&(G=JSON.parse(o.config))}catch(re){throw console.error("解析任务配置失败:",re),new Error("任务配置格式错误")}const q={taskId:o.id,taskName:o.name,sourceId:o.sourceId,targetId:o.targetId,syncDirection:te(o.sourceType,o.targetType),mysqlConfig:G.mysqlConfig,esConfig:G.esConfig,syncConfig:G.syncConfig||{threadCount:4,batchSize:1e3,errorStrategy:"skip",tableExistsStrategy:"drop"}};console.log("启动任务配置:",q),await X("start_sync",{config:q})}catch(o){throw g.value=`启动任务失败: ${o}`,console.error("startTask error:",o),o}finally{s.value=!1}}async function h(c){s.value=!0,g.value=null;try{await X("pause_sync",{taskId:c})}catch(o){throw g.value=`暂停任务失败: ${o}`,console.error("pauseTask error:",o),o}finally{s.value=!1}}async function S(c){s.value=!0,g.value=null;try{await X("resume_sync",{taskId:c})}catch(o){throw g.value=`恢复任务失败: ${o}`,console.error("resumeTask error:",o),o}finally{s.value=!1}}async function P(c){try{const o=await X("get_progress",{taskId:c});return t.value=o,o}catch(o){return g.value=`获取进度失败: ${o}`,console.error("getProgress error:",o),null}}async function V(c){try{const o=await X("get_errors",{taskId:c});return l.value=o,o}catch(o){return g.value=`获取错误日志失败: ${o}`,console.error("getErrors error:",o),[]}}async function z(c){try{return await X("get_task_logs",{taskId:c})}catch(o){return g.value=`获取任务日志失败: ${o}`,console.error("getTaskLogs error:",o),[]}}async function C(){await y();try{m=await me("task-progress",c=>{t.value=c.payload,console.log("Progress update:",c.payload)}),n=await me("task-error",c=>{l.value.push(c.payload),console.log("Error logged:",c.payload)}),console.log("Event listeners started")}catch(c){throw console.error("Failed to start event listeners:",c),c}}async function y(){m&&(m(),m=null),n&&(n(),n=null),console.log("Event listeners stopped")}function Y(){t.value=null,l.value=[]}function F(){g.value=null}function te(c,o){if(c==="mysql"&&o==="elasticsearch")return"MysqlToEs";if(c==="elasticsearch"&&o==="mysql")return"EsToMysql";if(c==="mysql"&&o==="mysql")return"MysqlToMysql";if(c==="elasticsearch"&&o==="elasticsearch")return"EsToEs";throw new Error(`不支持的同步方向: ${c} -> ${o}`)}return{progress:t,errors:l,loading:s,error:g,isRunning:v,isPaused:k,errorCount:I,startTask:M,pauseTask:h,resumeTask:S,getProgress:P,getErrors:V,getTaskLogs:z,startEventListeners:C,stopEventListeners:y,clearMonitorData:Y,clearError:F}}),Ft={class:"task-monitor"},qt={class:"task-list-header"},At={class:"task-list"},Dt=["onClick"],Vt={class:"task-card-header"},Yt={class:"task-name"},Wt={class:"task-card-info"},jt={class:"task-source"},Ut={style:{"font-size":"14px","font-weight":"500"}},Xt={style:{"font-size":"14px","font-weight":"500"}},Ht={style:{display:"flex","justify-content":"space-between"}},Gt={style:{"font-weight":"bold"}},Jt={key:0,style:{color:"#666","font-size":"13px"}},Kt={class:"table-progress-container"},Zt={class:"table-header"},Qt={class:"table-name"},ea={class:"table-percentage"},ta={class:"table-info"},aa={class:"log-container"},sa={class:"log-timestamp"},na={class:"log-level"},la={class:"log-message"},ra={key:0,class:"empty-log"},oa={class:"log-container"},ia={class:"log-timestamp"},ua={class:"log-level"},ca={class:"log-message"},da={key:0,class:"empty-log"},fa={class:"log-container"},ma={class:"log-timestamp"},va={class:"log-level"},ga={class:"log-message"},pa={key:0,class:"empty-log"},ha={class:"log-container"},ya={class:"error-header"},ba={class:"error-timestamp"},xa={class:"error-message"},_a={key:0,class:"error-details"},wa={key:0,class:"empty-log"},ka=H({__name:"TaskMonitor",setup(t){const l=dt(),s=Ot(),g=bt(),m=it(),n=$(""),v=$([]),k=N(()=>v.value.filter(e=>e.message.includes("批次")&&e.message.includes("完成")&&e.message.includes("已同步")&&e.message.includes("剩余"))),I=N(()=>v.value.filter(e=>e.message.includes("数据校验通过")||e.message.includes("数据校验失败"))),M=$(),h=$(),S=$(),P=$(),V=$();let z=null,C=null;const y=N(()=>n.value?s.progress:null),Y=N(()=>{var e;return((e=y.value)==null?void 0:e.status)==="running"}),F=N(()=>{var e;return((e=y.value)==null?void 0:e.status)==="paused"}),te=N(()=>{var e,u;return((e=y.value)==null?void 0:e.status)==="completed"?"success":((u=y.value)==null?void 0:u.status)==="failed"?"error":"info"});function c(){return l.tasks.find(e=>e.id===n.value)}function o(){const e=c();return e?g.dataSources.find(u=>u.id===e.sourceId):null}function G(){const e=c();return e?g.dataSources.find(u=>u.id===e.targetId):null}function q(e){return e===n.value?s.progress:null}function re(e){const u=g.dataSources.find(_=>_.id===e);return(u==null?void 0:u.name)||"未知数据源"}function Oe(e){return{running:"运行中",paused:"已暂停",completed:"已完成",failed:"失败"}[e||""]||"空闲"}function Fe(e){return{running:"info",paused:"warning",completed:"success",failed:"error"}[e||""]||"default"}function qe(e){return e==="completed"?"success":e==="failed"?"error":"info"}function Ae(e){return{waiting:"等待",running:"同步中",completed:"完成",failed:"失败"}[e]||e}function De(e){return{waiting:"default",running:"info",completed:"success",failed:"error"}[e]||"default"}function Ve(e){return e==="completed"?"success":e==="failed"?"error":e==="running"?"info":"default"}function ge(e){return{info:"INFO",warn:"WARN",error:"ERROR"}[e]||e.toUpperCase()}function Ye(e){if(!e||e<=0)return"-";const u=Math.floor(e/3600),_=Math.floor(e%3600/60),r=Math.floor(e%60);return u>0?`${u}h${_}m`:_>0?`${_}m${r}s`:`${r}s`}function We(e){n.value=e,v.value=[],pe(),oe(),he()}async function je(){if(n.value)try{await s.startTask(n.value),ye("任务已启动")}catch(e){ue(e,"启动任务失败")}}async function Ue(){if(n.value)try{await s.pauseTask(n.value),ye("任务已暂停")}catch(e){ue(e,"暂停任务失败")}}async function Xe(){if(n.value)try{await s.resumeTask(n.value),ye("任务已恢复")}catch(e){ue(e,"恢复任务失败")}}async function ke(){try{await l.fetchTasks()}catch(e){ue(e,"加载任务列表失败")}}async function pe(){if(n.value)try{await s.getProgress(n.value)}catch{}}async function oe(){if(n.value)try{await s.getErrors(n.value)}catch{}}async function he(){if(n.value)try{v.value=await s.getTaskLogs(n.value),await ee(),ae(h.value)}catch{}}async function He(){await ee(),M.value&&M.value.scrollIntoView({behavior:"smooth",block:"center"})}function ae(e){e&&(e.scrollTop=e.scrollHeight)}async function Ge(){try{z=await me("task-log",e=>{const u=e.payload;u.taskId===n.value&&(v.value.push(u.log),v.value.length>1e3&&v.value.shift(),ee(()=>{ae(h.value),ae(S.value),ae(P.value)}))}),C=await me("task-error",e=>{e.payload.taskId===n.value&&(oe(),ee(()=>ae(V.value)))})}catch(e){console.error("监听事件失败:",e)}}function Je(){z&&(z(),z=null),C&&(C(),C=null)}return Ie(async()=>{await g.fetchDataSources(),await ke(),await s.startEventListeners(),Ge();const e=m.query.taskId;e&&(n.value=e,pe(),oe(),he())}),we(()=>m.query.taskId,e=>{e&&typeof e=="string"&&(n.value=e,v.value=[],pe(),oe(),he())}),rt(()=>{s.stopEventListeners(),Je()}),(e,u)=>(p(),w("div",Ft,[d(a(ot),{"has-sider":"",style:{height:"100%"}},{default:i(()=>[d(a(ut),{bordered:"",width:320,"native-scrollbar":!1,style:{"background-color":"#fff"}},{default:i(()=>[f("div",qt,[u[0]||(u[0]=f("h3",null,"任务列表",-1)),d(a(ne),{size:"small",onClick:ke},{icon:i(()=>[d(a(U),null,{default:i(()=>[d(a(xt))]),_:1})]),_:1})]),f("div",At,[(p(!0),w(J,null,K(a(l).tasks,_=>{var r,T,W;return p(),w("div",{key:_.id,class:se(["task-card",{"is-selected":n.value===_.id}]),onClick:A=>We(_.id)},[f("div",Vt,[f("span",Yt,b(_.name),1),q(_.id)?(p(),B(a(be),{key:0,type:Fe((r=q(_.id))==null?void 0:r.status),size:"small"},{default:i(()=>{var A;return[O(b(Oe((A=q(_.id))==null?void 0:A.status)),1)]}),_:2},1032,["type"])):E("",!0)]),f("div",Wt,[f("span",jt,b(re(_.sourceId))+" → "+b(re(_.targetId)),1)]),q(_.id)?(p(),B(a(xe),{key:0,type:"line",percentage:((T=q(_.id))==null?void 0:T.percentage)||0,height:6,"show-indicator":!1,status:qe((W=q(_.id))==null?void 0:W.status)},null,8,["percentage","status"])):E("",!0)],10,Dt)}),128)),a(l).tasks.length===0?(p(),B(a(ze),{key:0,description:"暂无任务",style:{"margin-top":"40px"}})):E("",!0)])]),_:1}),d(a(ct),{"native-scrollbar":!1,style:{padding:"16px"}},{default:i(()=>[n.value?(p(),B(a(Z),{key:1,vertical:"",size:16},{default:i(()=>{var _;return[d(a(ce),{size:"small"},{default:i(()=>[d(a(Z),{align:"center",justify:"space-between"},{default:i(()=>[d(a(Z),{size:8,align:"center"},{default:i(()=>{var r,T,W,A;return[d(a(U),{size:24,color:((r=c())==null?void 0:r.sourceType)==="mysql"?"#2080f0":"#18a058"},{default:i(()=>{var j;return[(p(),B(Ce(((j=c())==null?void 0:j.sourceType)==="mysql"?a(Te):a($e))))]}),_:1},8,["color"]),f("span",Ut,b(((T=o())==null?void 0:T.name)||"-"),1),d(a(U),{size:20,color:"#999"},{default:i(()=>[d(a(ft))]),_:1}),d(a(U),{size:24,color:((W=c())==null?void 0:W.targetType)==="mysql"?"#2080f0":"#18a058"},{default:i(()=>{var j;return[(p(),B(Ce(((j=c())==null?void 0:j.targetType)==="mysql"?a(Te):a($e))))]}),_:1},8,["color"]),f("span",Xt,b(((A=G())==null?void 0:A.name)||"-"),1)]}),_:1}),d(a(Z),null,{default:i(()=>[d(a(ne),{type:"primary",disabled:Y.value||F.value,loading:a(s).loading,onClick:je},{icon:i(()=>[d(a(U),null,{default:i(()=>[d(a(Ee))]),_:1})]),default:i(()=>[u[1]||(u[1]=O(" 启动 ",-1))]),_:1},8,["disabled","loading"]),d(a(ne),{type:"warning",disabled:!Y.value,loading:a(s).loading,onClick:Ue},{icon:i(()=>[d(a(U),null,{default:i(()=>[d(a(Mt))]),_:1})]),default:i(()=>[u[2]||(u[2]=O(" 暂停 ",-1))]),_:1},8,["disabled","loading"]),d(a(ne),{type:"info",disabled:!F.value,loading:a(s).loading,onClick:Xe},{icon:i(()=>[d(a(U),null,{default:i(()=>[d(a(Ee))]),_:1})]),default:i(()=>[u[3]||(u[3]=O(" 恢复 ",-1))]),_:1},8,["disabled","loading"])]),_:1}),d(a(Z),null,{default:i(()=>{var r,T;return[d(a(Re),{label:"同步速度",value:((r=y.value)==null?void 0:r.speed.toFixed(2))||0},{suffix:i(()=>[...u[4]||(u[4]=[O("记录/秒",-1)])]),_:1},8,["value"]),d(a(Re),{label:"预计剩余",value:Ye(((T=y.value)==null?void 0:T.estimatedTime)||0)},null,8,["value"])]}),_:1})]),_:1})]),_:1}),d(a(ce),{size:"small",title:"同步进度"},{default:i(()=>[d(a(Z),{vertical:"",size:12},{default:i(()=>{var r,T,W,A,j;return[f("div",Ht,[f("span",null,b(((r=y.value)==null?void 0:r.processedRecords.toLocaleString())||0)+" / "+b(((T=y.value)==null?void 0:T.totalRecords.toLocaleString())||0),1),f("span",Gt,b(((W=y.value)==null?void 0:W.percentage.toFixed(2))||0)+"%",1)]),d(a(xe),{type:"line",percentage:((A=y.value)==null?void 0:A.percentage)||0,status:te.value,height:24},null,8,["percentage","status"]),(j=y.value)!=null&&j.currentTable?(p(),w("div",Jt," 当前表："+b(y.value.currentTable),1)):E("",!0)]}),_:1})]),_:1}),(_=y.value)!=null&&_.tableProgress&&y.value.tableProgress.length>0?(p(),B(a(ce),{key:0,size:"small",title:"表同步进度"},{"header-extra":i(()=>[d(a(ne),{size:"small",onClick:He},{default:i(()=>[...u[5]||(u[5]=[O(" 定位当前表 ",-1)])]),_:1})]),default:i(()=>[f("div",Kt,[(p(!0),w(J,null,K(y.value.tableProgress,r=>(p(),w("div",{key:r.tableName,class:se(["table-progress-item",{"is-current":r.status==="running"}]),ref_for:!0,ref:r.status==="running"?"currentTableRef":void 0},[f("div",Zt,[d(a(be),{type:De(r.status),size:"small",style:{"margin-right":"8px"}},{default:i(()=>[O(b(Ae(r.status)),1)]),_:2},1032,["type"]),f("span",Qt,b(r.tableName),1),f("span",ea,b(r.percentage.toFixed(1))+"%",1)]),d(a(xe),{type:"line",percentage:r.percentage,status:Ve(r.status),height:6,"show-indicator":!1},null,8,["percentage","status"]),f("div",ta,b(r.processedRecords.toLocaleString())+" / "+b(r.totalRecords.toLocaleString()),1)],2))),128))])]),_:1})):E("",!0),d(a(ce),{size:"small"},{default:i(()=>[d(a(wt),{type:"line",animated:""},{default:i(()=>[d(a(de),{name:"logs",tab:"实时日志"},{tab:i(()=>[u[6]||(u[6]=O(" 实时日志 ",-1)),v.value.length>0?(p(),B(a(fe),{key:0,value:v.value.length,max:999,style:{"margin-left":"8px"}},null,8,["value"])):E("",!0)]),default:i(()=>[f("div",aa,[f("div",{ref_key:"allLogContentRef",ref:h,class:"log-content"},[(p(!0),w(J,null,K(v.value,(r,T)=>(p(),w("div",{key:T,class:se(["log-entry",`log-${r.level}`])},[f("span",sa,b(r.timestamp),1),f("span",na,b(ge(r.level)),1),f("span",la,b(r.message),1)],2))),128)),v.value.length===0?(p(),w("div",ra," 暂无日志 ")):E("",!0)],512)])]),_:1}),d(a(de),{name:"detail",tab:"明细日志"},{tab:i(()=>[u[7]||(u[7]=O(" 明细日志 ",-1)),k.value.length>0?(p(),B(a(fe),{key:0,value:k.value.length,max:999,style:{"margin-left":"8px"}},null,8,["value"])):E("",!0)]),default:i(()=>[f("div",oa,[f("div",{ref_key:"detailLogContentRef",ref:S,class:"log-content"},[(p(!0),w(J,null,K(k.value,(r,T)=>(p(),w("div",{key:T,class:se(["log-entry",`log-${r.level}`])},[f("span",ia,b(r.timestamp),1),f("span",ua,b(ge(r.level)),1),f("span",ca,b(r.message),1)],2))),128)),k.value.length===0?(p(),w("div",da," 暂无明细日志 ")):E("",!0)],512)])]),_:1}),d(a(de),{name:"verify",tab:"校验日志"},{tab:i(()=>[u[8]||(u[8]=O(" 校验日志 ",-1)),I.value.length>0?(p(),B(a(fe),{key:0,value:I.value.length,max:999,style:{"margin-left":"8px"}},null,8,["value"])):E("",!0)]),default:i(()=>[f("div",fa,[f("div",{ref_key:"verifyLogContentRef",ref:P,class:"log-content"},[(p(!0),w(J,null,K(I.value,(r,T)=>(p(),w("div",{key:T,class:se(["log-entry",`log-${r.level}`])},[f("span",ma,b(r.timestamp),1),f("span",va,b(ge(r.level)),1),f("span",ga,b(r.message),1)],2))),128)),I.value.length===0?(p(),w("div",pa," 暂无校验日志 ")):E("",!0)],512)])]),_:1}),d(a(de),{name:"errors",tab:"错误日志"},{tab:i(()=>[u[9]||(u[9]=O(" 错误日志 ",-1)),a(s).errors.length>0?(p(),B(a(fe),{key:0,value:a(s).errors.length,max:99,type:"error",style:{"margin-left":"8px"}},null,8,["value"])):E("",!0)]),default:i(()=>[f("div",ha,[f("div",{ref_key:"errorLogContentRef",ref:V,class:"log-content"},[(p(!0),w(J,null,K(a(s).errors,(r,T)=>(p(),w("div",{key:T,class:"error-entry"},[f("div",ya,[f("span",ba,b(new Date(r.timestamp).toLocaleString()),1),d(a(be),{type:"error",size:"small"},{default:i(()=>[O(b(r.errorType),1)]),_:2},1024)]),f("div",xa,b(r.message),1),r.data?(p(),w("div",_a,[d(a(mt),null,{default:i(()=>[d(a(vt),{title:"详细信息"},{default:i(()=>[f("pre",null,b(JSON.stringify(r.data,null,2)),1)]),_:2},1024)]),_:2},1024)])):E("",!0)]))),128)),a(s).errors.length===0?(p(),w("div",wa," 暂无错误 ")):E("",!0)],512)])]),_:1})]),_:1})]),_:1})]}),_:1})):(p(),B(a(ze),{key:0,description:"请从左侧选择一个任务",style:{"margin-top":"100px"}}))]),_:1})]),_:1})]))}}),za=_t(ka,[["__scopeId","data-v-728d09e1"]]);export{za as default};
