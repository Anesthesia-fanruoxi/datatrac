import { createApp } from 'vue'
import { createPinia } from 'pinia'
import router from './router'
import App from './App.vue'

// 导入 Naive UI 的通用样式
import 'vfonts/Lato.css'
import 'vfonts/FiraCode.css'

const app = createApp(App)
const pinia = createPinia()

app.use(pinia)
app.use(router)
app.mount('#app')
