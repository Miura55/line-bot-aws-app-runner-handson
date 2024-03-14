#!/usr/bin/env node
import 'source-map-support/register';
import * as cdk from 'aws-cdk-lib';
import { AppStack } from '../lib/app-stack';
import { DeployStack } from '../lib/deploy-stack';

const app = new cdk.App();

const deployStack = new DeployStack(app, 'HandsonDeployStack', {
  env: {
    account: process.env.CDK_DEFAULT_ACCOUNT,
    region: 'ap-northeast-1',
  },
});

new AppStack(app, 'HandsonAppStack', {
  env: {
    account: process.env.CDK_DEFAULT_ACCOUNT,
    region: 'ap-northeast-1',
  },
  ecrRepository: deployStack.ecrRepository,
});
