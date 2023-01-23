/*
 Copyright (c) 2022 NTT Communications Corporation

 Permission is hereby granted, free of charge, to any person obtaining a copy
 of this software and associated documentation files (the "Software"), to deal
 in the Software without restriction, including without limitation the rights
 to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 copies of the Software, and to permit persons to whom the Software is
 furnished to do so, subject to the following conditions:

 The above copyright notice and this permission notice shall be included in
 all copies or substantial portions of the Software.

 THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 THE SOFTWARE.
*/

package gitrepo

import "github.com/shurcooL/githubv4"

type healthQuery struct {
	Viewer struct {
		Login githubv4.String
	}
}

type getRepoQuery struct {
	Repository struct {
		ID githubv4.String
	} `graphql:"repository(owner:$repositoryOwner,name:$repositoryName)"`
}

func getRepoVariable(repo GitRepoRef) map[string]interface{} {
	return map[string]interface{}{
		"repositoryOwner": githubv4.String(repo.Owner),
		"repositoryName":  githubv4.String(repo.Name),
	}
}

type createPullRequestMutation struct {
	CreatePullRequest struct {
		PullRequest struct {
			Number int
		}
	} `graphql:"createPullRequest(input:$input)"`
}

func createPullRequestInput(repoID githubv4.ID, payload GitPullRequestPayload) githubv4.CreatePullRequestInput {
	return githubv4.CreatePullRequestInput{
		RepositoryID: repoID,
		BaseRefName:  githubv4.String(payload.BaseRef),
		HeadRefName:  githubv4.String(payload.HeadRef),
		Title:        githubv4.String(payload.Title),
		Body:         githubv4.NewString(githubv4.String(payload.Body)),
	}
}
