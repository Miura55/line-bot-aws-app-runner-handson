#!/usr/bin/env node
import 'source-map-support/register';
import * as cdk from 'aws-cdk-lib';
import { AppStack } from '../lib/app-stack';
import { DeployStack } from '../lib/deploy-stack';
import { PipelineStack } from '../lib/pipeline-stack';

const app = new cdk.App();
const region = 'ap-northeast-1';

const deployStack = new DeployStack(app, 'HandsonDeployStack', {
  env: {
    account: process.env.CDK_DEFAULT_ACCOUNT,
    region: region,
  },
});

const appStack = new AppStack(app, 'HandsonAppStack', {
  env: {
    account: process.env.CDK_DEFAULT_ACCOUNT,
    region: region,
  },
  ecrRepository: deployStack.ecrRepository,
});

new PipelineStack(app, 'HandsonPipelineStack', {
  env: {
    account: process.env.CDK_DEFAULT_ACCOUNT,
    region: region,
  },
  ecrRepository: deployStack.ecrRepository,
});
