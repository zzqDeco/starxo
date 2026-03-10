/// <reference types="vite/client" />

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<{}, {}, any>
  export default component
}

interface Window {
  go?: {
    main?: {
      ChatService?: {
        SendMessage(content: string, filePath?: string): Promise<void>
        StopGeneration(): Promise<void>
      }
      SettingsService?: {
        GetSettings(): Promise<import('./types/config').AppSettings>
        SaveSettings(settings: import('./types/config').AppSettings): Promise<void>
        TestLLM(config: import('./types/config').LLMConfig): Promise<boolean>
      }
      ConnectionService?: {
        Connect(): Promise<void>
        Disconnect(): Promise<void>
        TestSSH(config: import('./types/config').SSHConfig): Promise<boolean>
      }
      FileService?: {
        ListFiles(): Promise<import('./types/config').FileInfo[]>
        UploadFile(name: string, data: number[]): Promise<void>
        DownloadFile(path: string): Promise<void>
      }
    }
  }
  runtime?: {
    OpenFileDialog(options: { title?: string }): Promise<string>
  }
}
