import { describe, expect, it } from 'vitest'
import { anyttyI18n } from '../i18n'
import { basename, fileEntryMeta, fileEntryPath, fileEntryMenuSubtitle, joinPath, normalizeFilePath, parentPath, pathBreadcrumbs } from './fileUtils'

describe('file path utilities', () => {
  it('normalizes and navigates Windows drive paths', () => {
    expect(normalizeFilePath('C:\\Users\\Ada\\')).toBe('C:/Users/Ada')
    expect(joinPath('C:\\Users\\Ada', 'Documents')).toBe('C:/Users/Ada/Documents')
    expect(parentPath('C:\\Users\\Ada')).toBe('C:/Users')
    expect(parentPath('C:\\')).toBe('C:/')
    expect(basename('C:\\Users\\Ada\\notes.txt')).toBe('notes.txt')
  })

  it('keeps UNC share roots stable while navigating', () => {
    expect(normalizeFilePath('\\\\server\\share\\folder\\')).toBe('//server/share/folder')
    expect(parentPath('//server/share/folder')).toBe('//server/share')
    expect(parentPath('//server/share')).toBe('//server/share')
  })

  it('builds navigable breadcrumbs without changing the path root kind', () => {
    expect(pathBreadcrumbs('/var/log')).toEqual([
      { label: '/', path: '/' },
      { label: 'var', path: '/var' },
      { label: 'log', path: '/var/log' },
    ])
    expect(pathBreadcrumbs('C:\\Users\\lozzow')).toEqual([
      { label: '/', path: '/' },
      { label: 'C:', path: 'C:/' },
      { label: 'Users', path: 'C:/Users' },
      { label: 'lozzow', path: 'C:/Users/lozzow' },
    ])
    expect(pathBreadcrumbs('\\\\server\\share\\folder')).toEqual([
      { label: '//server/share', path: '//server/share' },
      { label: 'folder', path: '//server/share/folder' },
    ])
  })

  it('uses an entry absolute path for virtual Windows drive roots', () => {
    expect(fileEntryPath('/', { name: 'C:', path: 'C:/' })).toBe('C:/')
    expect(fileEntryPath('/home', { name: 'ada' })).toBe('/home/ada')
  })

  it('localizes visible file metadata without translating technical values', async () => {
    await anyttyI18n.changeLanguage('zh-CN')
    try {
      expect(fileEntryMeta({ type: 'dir', size: 0, childCount: 2 })).toBe('2 个项目')
      expect(fileEntryMenuSubtitle({
        type: 'file',
        size: 128,
        hardLink: true,
        linkCount: 2,
        inode: 42,
      })).toBe('128 B · 硬链接 x2 · inode 42')
    } finally {
      await anyttyI18n.changeLanguage('en')
    }
  })
})
