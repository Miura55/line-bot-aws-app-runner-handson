import { StackProps, Stack, Duration, RemovalPolicy } from 'aws-cdk-lib';
import { Construct } from 'constructs';
import { Role } from 'aws-cdk-lib/aws-iam';
import { Table, AttributeType } from 'aws-cdk-lib/aws-dynamodb';

interface DatabaseStackProps extends StackProps {
  tableName: string;
  appRunnerInstanceRole: Role;
}

export class DatabaseStack extends Stack {
  constructor(scope: Construct, id: string, props: DatabaseStackProps) {
    super(scope, id, props);

    const tableName = props.tableName;
    const dynamodbTable = new Table(this, 'LineBotHandsonTable', {
      partitionKey: {
        name: 'userId',
        type: AttributeType.STRING,
      },
      sortKey: {
        name: 'timestamp',
        type: AttributeType.STRING,
      },
      removalPolicy: RemovalPolicy.DESTROY,
      tableName: tableName,
    });
    dynamodbTable.applyRemovalPolicy(RemovalPolicy.DESTROY);

    // App RunnerのインスタンスロールにDynamoDBテーブルへのアクセス権限を付与
    dynamodbTable.grantReadWriteData(props.appRunnerInstanceRole);
  }
}
