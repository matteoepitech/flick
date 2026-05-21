export type SettingFieldType = "text" | "textarea" | "number" | "switch" | "select"

export type SettingOption = { value: string; label: string }

export type SettingDependency = {
  key: string
  equals: string | number | boolean
}

export type SettingField = {
  key: string
  type: SettingFieldType
  defaultValue: string | number | boolean
  placeholder?: string
  options?: SettingOption[]
  hasDescription?: boolean
  dependsOn?: SettingDependency
  notAvailable?: boolean
}

export type SettingSection = {
  id: string
  hasDescription?: boolean
  fields: SettingField[]
}

export const settingsSections: SettingSection[] = [
  {
    id: "storage",
    hasDescription: true,
    fields: [
      {
        key: "persistence",
        type: "switch",
        defaultValue: true,
        hasDescription: true,
        notAvailable: true,
      },
    ],
  },
  {
    id: "uploads",
    hasDescription: true,
    fields: [
      {
        key: "max_file_size_mb",
        type: "number",
        defaultValue: 1000,
      },
      {
        key: "default_expiration",
        type: "text",
        defaultValue: "15m",
        hasDescription: true,
      },
      {
        key: "max_expiration",
        type: "text",
        defaultValue: "4h",
      },
      {
        key: "allow_multiple_downloads",
        type: "switch",
        defaultValue: false,
        hasDescription: true,
      },
      {
        key: "default_download_count",
        type: "number",
        defaultValue: 1,
        dependsOn: { key: "allow_multiple_downloads", equals: true },
        hasDescription: true,
      },
      {
        key: "max_download_count",
        type: "number",
        defaultValue: 5,
        dependsOn: { key: "allow_multiple_downloads", equals: true },
      },
    ],
  },
  {
    id: "security",
    hasDescription: true,
    fields: [
      {
        key: "require_password",
        type: "switch",
        defaultValue: false,
        hasDescription: true,
        notAvailable: true,
      },
    ],
  },
  {
    id: "anti_abuse",
    hasDescription: true,
    fields: [
      {
        key: "activate_rate_limit",
        type: "switch",
        defaultValue: true,
        hasDescription: true,
      },
      {
        key: "max_generation_key_per_hour",
        type: "number",
        defaultValue: 60,
        dependsOn: { key: "activate_rate_limit", equals: true },
        hasDescription: true,
      },
      {
        key: "max_upload_per_hour_per_key",
        type: "number",
        defaultValue: 10,
        dependsOn: { key: "activate_rate_limit", equals: true },
        hasDescription: true,
      },
      {
        key: "max_upload_per_hour_per_ip",
        type: "number",
        defaultValue: 30,
        dependsOn: { key: "activate_rate_limit", equals: true },
        hasDescription: true,
      },
      {
        key: "max_upload_per_hour",
        type: "number",
        defaultValue: 1000,
        dependsOn: { key: "activate_rate_limit", equals: true },
        hasDescription: true,
      },
    ],
  },
]
