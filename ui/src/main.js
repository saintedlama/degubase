import { createApp } from 'vue'
import 'remixicon/fonts/remixicon.css'
import './style.css'
import App from './App.vue'
import router from './router/index.js'

createApp(App).use(router).mount('#app')
