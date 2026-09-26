#!/usr/bin/env python3
"""Build a portable per-user Mac setup app. Requires macOS Xcode command-line tools."""
import argparse, os, pathlib, plistlib, shutil, subprocess
p=argparse.ArgumentParser();p.add_argument('--arch',choices=['arm64','amd64'],required=True);p.add_argument('--output',type=pathlib.Path,required=True);a=p.parse_args()
repo=pathlib.Path(__file__).resolve().parents[1];out=a.output.resolve();app=out/'AI Usage Monitor.app';contents=app/'Contents';resources=contents/'Resources';macos=contents/'MacOS'
resources.mkdir(parents=True,exist_ok=True);macos.mkdir(parents=True,exist_ok=True)
env={**os.environ,'CGO_ENABLED':'0','GOOS':'darwin','GOARCH':a.arch}
for name in ['team-agent','team-setup']:
 subprocess.run(['go','build','-trimpath','-o',str(resources/name),'./cmd/'+name],cwd=repo,env=env,check=True)
arch='x86_64' if a.arch=='amd64' else 'arm64'
subprocess.run(['xcrun','swiftc','-O','-target',arch+'-apple-macos13.0','-framework','AppKit',str(repo/'desktop/macos/App.swift'),'-o',str(macos/'AIUsageMonitor')],check=True)
info={'CFBundleName':'AI Usage Monitor','CFBundleDisplayName':'Shared Account AI Usage Monitor','CFBundleIdentifier':'org.sharedaccountmonitor.setup','CFBundleExecutable':'AIUsageMonitor','CFBundlePackageType':'APPL','CFBundleShortVersionString':'0.5.0','CFBundleVersion':'6','LSMinimumSystemVersion':'13.0','NSHighResolutionCapable':True,'CFBundleDocumentTypes':[{'CFBundleTypeName':'AI Usage Monitor Connection','CFBundleTypeRole':'Viewer','LSHandlerRank':'Owner','LSItemContentTypes':['org.sharedaccountmonitor.connection']}],'UTExportedTypeDeclarations':[{'UTTypeIdentifier':'org.sharedaccountmonitor.connection','UTTypeDescription':'Private workspace connection','UTTypeConformsTo':['public.json'],'UTTypeTagSpecification':{'public.filename-extension':['aiusage']}}]}
(contents/'Info.plist').write_bytes(plistlib.dumps(info))
shutil.copy2(repo/'LICENSE',resources/'LICENSE');shutil.copy2(repo/'THIRD_PARTY_NOTICES.md',resources/'THIRD_PARTY_NOTICES.md')
subprocess.run(['python3',str(repo/'scripts/dependency_notices.py'),str(resources/'dependency-notices')],check=True)
# Ad-hoc integrity signing is not Developer ID signing/notarization. Never claim otherwise.
for binary in [resources/'team-agent',resources/'team-setup',macos/'AIUsageMonitor']:
 subprocess.run(['codesign','--force','--sign','-',str(binary)],check=True)
subprocess.run(['codesign','--force','--sign','-',str(app)],check=True)
subprocess.run(['codesign','--verify','--deep','--strict',str(app)],check=True)
zipfile=out/f'AI-Usage-Monitor-mac-{a.arch}.zip'
subprocess.run(['ditto','-c','-k','--sequesterRsrc','--keepParent',str(app),str(zipfile)],check=True)
print(zipfile)
