import { createApp } from 'vue'
import { createPinia } from 'pinia'
import Antd from 'ant-design-vue'
import 'ant-design-vue/dist/reset.css'
import { Capacitor } from '@capacitor/core'
import { StatusBar } from '@capacitor/status-bar'
import { App as CapacitorApp } from '@capacitor/app'

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

let androidBackHandlerRegistered = false

const configureAndroidBackNavigation = () => {
  if (androidBackHandlerRegistered || Capacitor.getPlatform() !== 'android') {
    return
  }

  androidBackHandlerRegistered = true

  CapacitorApp.addListener('backButton', ({ canGoBack }) => {
    const currentRoute = router.currentRoute.value
    const currentRouteName = currentRoute.name as string | undefined

    if (
      currentRouteName === 'home' ||
      currentRouteName === 'login' ||
      currentRouteName === 'register'
    ) {
      CapacitorApp.exitApp()
      return
    }

    if (canGoBack && window.history.length > 1) {
      window.history.back()
      return
    }

    if (currentRoute.name !== 'home') {
      router.push({ name: 'home' })
      return
    }

    CapacitorApp.exitApp()
  }).catch((error) => {
    console.warn('[BackButton] Failed to register Android back handler', error)
  })
}

configureAndroidBackNavigation()
