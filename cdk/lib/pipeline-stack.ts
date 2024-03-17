import * as path from 'path';
import { StackProps, Stack, RemovalPolicy } from 'aws-cdk-lib';
import { Construct } from 'constructs';
import * as codecommit from 'aws-cdk-lib/aws-codecommit';
import { LogGroup } from 'aws-cdk-lib/aws-logs';
import { PolicyStatement } from 'aws-cdk-lib/aws-iam';
import { Bucket } from 'aws-cdk-lib/aws-s3';
import { Artifact, Pipeline, PipelineType } from 'aws-cdk-lib/aws-codepipeline';
import { CodeCommitSourceAction, CodeBuildAction, CodeCommitTrigger } from 'aws-cdk-lib/aws-codepipeline-actions';
import { Project, Source, LinuxBuildImage, ComputeType, BuildSpec } from 'aws-cdk-lib/aws-codebuild';
import { Repository } from 'aws-cdk-lib/aws-ecr';

export interface PipelineStackProps extends StackProps {
  ecrRepository: Repository;
}

export class PipelineStack extends Stack {
  constructor(scope: Construct, id: string, props?: PipelineStackProps) {
    super(scope, id, props);

    // CodeCommitのレポジトリを作成
    const repository = new codecommit.Repository(this, 'Repository', {
      repositoryName: 'line-bot-hands-on',
      code: codecommit.Code.fromDirectory(
        path.join(__dirname, '../../app'),
        'main'
      )
    });

    // LogGroupを作成
    const logGroup = new LogGroup(this, 'LogGroup', {
      logGroupName: '/aws/codebuild/line-bot-hand/apps-on',
      removalPolicy: RemovalPolicy.DESTROY,
    });
    logGroup.applyRemovalPolicy(RemovalPolicy.DESTROY);

    // CodeBuildのプロジェクトを作成
    const codeBuildProject = new Project(this, 'CodeBuildProject', {
      projectName: 'line-bot-hands-on',
      source: Source.codeCommit({ 
        repository: repository,
      }),
      environment: {
        buildImage: LinuxBuildImage.STANDARD_7_0,
        computeType: ComputeType.SMALL,
        privileged: true,
        environmentVariables: {
          AWS_ACCOUNT_ID: { value: this.account },
          AWS_REGION: { value: this.region },
          ECR_REPOSITORY_URI: { value: props?.ecrRepository.repositoryUri },
        },
      },
      logging: {
        cloudWatch: {
          logGroup: logGroup,
          enabled: true,
        },
      },
      buildSpec: BuildSpec.fromObject({
        version: '0.2',
        phases: {
          pre_build: {
            commands: [
              "echo Logging in to Amazon ECR...",
              "aws --version",
              "aws ecr get-login-password --region $AWS_DEFAULT_REGION | docker login --username AWS --password-stdin $AWS_ACCOUNT_ID.dkr.ecr.$AWS_DEFAULT_REGION.amazonaws.com",
              "COMMIT_HASH=$(echo $CODEBUILD_RESOLVED_SOURCE_VERSION | cut -c 1-7)",
              "IMAGE_TAG=${COMMIT_HASH:=latest}",
            ],
          },
          build: {
            commands: [
              "echo Build started on `date`",
              "echo Building the Docker image...",
              "docker build -t $ECR_REPOSITORY_URI:latest .",
              "docker tag $ECR_REPOSITORY_URI:latest $ECR_REPOSITORY_URI:$IMAGE_TAG",
              "docker tag $ECR_REPOSITORY_URI:latest $ECR_REPOSITORY_URI:latest",
            ],
          },
          post_build: {
            commands: [
              "echo Build completed on `date`",
              "echo Pushing the Docker images...",
              "docker push $ECR_REPOSITORY_URI:latest",
              "docker push $ECR_REPOSITORY_URI:$IMAGE_TAG",
              "echo Writing image definitions file...",
              "printf '[{\"name\":\"line-bot-hands-on\",\"imageUri\":\"%s\"}]' $ECR_REPOSITORY_URI:$IMAGE_TAG > imagedefinitions.json",
            ],
          },
        },
        artifacts: {
          files : [
            "imagedefinitions.json"
          ],
        },
      }),
    });
    
    // ECRにアクセスするためのIAMポリシーを作成
    const ecrPolicy = new PolicyStatement({
      actions: [
        "ecr:BatchCheckLayerAvailability",
        "ecr:CompleteLayerUpload",
        "ecr:GetAuthorizationToken",
        "ecr:InitiateLayerUpload",
        "ecr:PutImage",
        "ecr:UploadLayerPart",
      ],
      resources: ['*'],
    });
    
    // CodeBuildにECRのポリシーをアタッチ
    codeBuildProject.addToRolePolicy(ecrPolicy);

    // S3バケットを作成
    const artifactBucket = new Bucket(this, 'ArtifactBucket', {
      removalPolicy: RemovalPolicy.DESTROY,
      autoDeleteObjects: true,
    });
    artifactBucket.applyRemovalPolicy(RemovalPolicy.DESTROY);

    // CodePipelineを作成
    const pipeline = new Pipeline(this, 'Pipeline', {
      pipelineName: 'line-bot-hands-on',
      artifactBucket: artifactBucket,
      pipelineType: PipelineType.V2,
    });

    // sourceステージを追加
    const sourceOutput = new Artifact();
    const sourceAction = new CodeCommitSourceAction({
      actionName: 'CodeCommit',
      repository: repository,
      output: sourceOutput,
      branch: 'main',
      trigger: CodeCommitTrigger.EVENTS,
    });
    pipeline.addStage({
      stageName: 'Source',
      actions: [sourceAction],
    });
    
    // buildステージを追加
    const buildOutput = new Artifact();
    const buildAction = new CodeBuildAction({
      actionName: 'CodeBuild',
      project: codeBuildProject,
      input: sourceOutput,
      outputs: [buildOutput],
    });
    pipeline.addStage({
      stageName: 'Build',
      actions: [buildAction],
    });
  }
}
