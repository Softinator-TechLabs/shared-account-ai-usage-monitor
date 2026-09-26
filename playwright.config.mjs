import {defineConfig} from '@playwright/test';
export default defineConfig({testDir:'./tests/browser',workers:1,use:{baseURL:process.env.TEST_ORIGIN||'http://127.0.0.1:18090',viewport:{width:1440,height:1000}},reporter:'list'});
