#!/bin/bash

echo "Using the following as bad versions for input: "
ls multiple_bad_versions

echo "Using the following as a good version: "
ls good_version

echo "Picking this version to build in to archives: "
ls multiple_bad_versions/log4j-core-2.14.1.jar

bad_version=multiple_bad_versions/log4j-core-2.14.1.jar
bad_version_name=log4j-core-2.14.1.jar

mkdir tmp

echo "Creating single_bad_version with a single bad log4j jar inside"

mkdir -p single_bad_version
cp -f $bad_version single_bad_version/

echo "Done"

echo "Creating inside_a_dist with a bad log4j jar inside archives"

mkdir -p inside_a_dist
rm -f inside_a_dist/*

cd multiple_bad_versions
tar -czvf ../inside_a_dist/wrapped_log4j.tar.gz $bad_version_name
zip ../inside_a_dist/wrapped_log4j.zip $bad_version_name
cd ../

echo "Done"

echo "Creating a par with log4j in the lib directory"

mkdir -p inside_a_par
rm -f inside_a_par/*

mkdir -p tmp/lib
cp $bad_version tmp/lib
mkdir -p tmp/classes/some/other
touch tmp/classes/some/other/File.class
cd tmp
zip -r ../inside_a_par/wrapped_in_a_par.par *
cd ..

rm -rf tmp/*

echo "Done"

echo "Creating a dist with the bad par in"

mkdir -p par_in_a_dist
rm -r par_in_a_dist/*

cd inside_a_par
zip ../par_in_a_dist/wrapped_par_in_a_dist.zip *
cd ..

echo "Done"

echo "Creating a fat jar"

mkdir -p fat_jar
rm -f fat_jar/*

mkdir -p tmp
cp $bad_version tmp/

cd tmp
unzip $bad_version_name
rm $bad_version_name
# Add files to the jar to make it "fat"
mkdir -p some/other/
mkdir -p yet/another/
touch some/other/File.class
touch yet/another/Clazz.class
zip -r ../fat_jar/fat_jar.jar *
cd ..
rm -rf tmp/*

echo "Done"

echo "Creating a dist with the fat jar in"

mkdir -p archived_fat_jar
rm -f archived_fat_jar/*

cd fat_jar
tar -cvzf ../archived_fat_jar/archived_fat_jar.tar.gz *
cd ..

echo "Done"

echo "Building a lightly shaded jar"

mkdir -p light_shading
rm -f light_shading/*

cd java_projects/light_shading
./gradlew build
cp app/build/libs/shadow-all.jar ../../light_shading/shadow-all.jar
cd ../..

echo "Done"

rm -rf tmp
