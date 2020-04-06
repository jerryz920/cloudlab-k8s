bin/spark-submit \
  --master k8s://https://10.10.2.1:6443 \
  --deploy-mode cluster \
  --name spark-pi \
  --class org.apache.spark.examples.SparkPi \
  --conf spark.kubernetes.namespace=latte-alice \
  --conf spark.executor.instances=5 \
  --conf spark.latte.outputTag=testtag \
  --conf spark.kubernetes.container.image=spark:v2.3 \
  --conf spark.kubernetes.authenticate.submission.caCertFile=/var/run/secrets/kubernetes.io/serviceaccount/ca.crt \
  --conf spark.kubernetes.authenticate.driver.serviceAccountName=latte-admin-alice \
  --conf spark.kubernetes.driver.pod.name=spark-pi-driver \
   "http://192.168.0.1:12345/spark-examples_2.11-2.3.2.jar"

