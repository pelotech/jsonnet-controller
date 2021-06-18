/*
Copyright 2021 Pelotech.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

{
    // semver(version): parse the version string into an object containing
    // the following fields:
    //   
    //   - major: The major value
    //   - minor: The minor value
    //   - patch: The patch value
    //   - pre_release: The pre-releae value
    //   - metadata: Any metadata on the version string
    //
    // For example:
    //
    //   { major_version: pelotech.semver('1.2.3').major }
    //
    // Would produce:
    // 
    //   { "major_version": 1 }
    //
    semver:: std.native('semver'),

    // semverCompare(constraint, version): Compares the given version against
    // the provided constraint. For more information, see the sprig documentation.
    // https://masterminds.github.io/sprig/semver.html
    semverCompare:: std.native('semverCompare'),

    // sha1Sum(string): Computes a SHA1 sum of the given string.
    sha1Sum:: std.native('sha1Sum'),

    // sha256Sum(string): Computes a SHA256 sum of the given string.
    sha256Sum:: std.native('sha1Sum'),
}