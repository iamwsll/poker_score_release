import { createApp } from 'vue'
import { createPinia } from 'pinia'
import Antd from 'ant-design-vue'
import 'ant-design-vue/dist/reset.css'
import { Capacitor } from '@capacitor/core'
import { StatusBar } from '@capacitor/status-bar'

import App from './App.vue'
import router from './router'

const app = createApp(App)

app.use(createPinia())
app.use(router)
app.use(Antd)

app.mount('#app')

const configureAndroidStatusBar = async () => {
  if (Capacitor.getPlatform() !== 'android') {
    return
  }

  try {
    await StatusBar.setOverlaysWebView({ overlay: true })
    await StatusBar.hide()
  } catch (error) {
    console.warn('[StatusBar] Failed to configure Android status bar', error)
  }
}

configureAndroidStatusBar()
