package cli_test

import (
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/client/flags"
	clitestutil "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/regen-network/bec/app"
	"github.com/regen-network/bec/testutil/network"
	"github.com/regen-network/bec/x/blog"
	"github.com/regen-network/bec/x/blog/client/cli"
)

type IntegrationTestSuite struct {
	suite.Suite

	app *app.RegenApp
	ctx sdk.Context

	cfg     network.Config
	network *network.Network
}

func (s *IntegrationTestSuite) SetupSuite() {
	s.T().Log("setting up integration test suite")

	cfg := network.DefaultConfig()
	cfg.NumValidators = 1

	s.cfg = cfg
	s.network = network.New(s.T(), cfg)

	_, err := s.network.WaitForHeight(1)
	s.Require().NoError(err)
}

func (s *IntegrationTestSuite) TearDownSuite() {
	s.T().Log("tearing down integration test suite")
	s.network.Cleanup()
}

func (s *IntegrationTestSuite) TestCreatePost() {
	val0 := s.network.Validators[0]

	testCases := []struct {
		name      string
		args      []string
		expErr    bool
		expErrMsg string
	}{
		{
			"no author",
			[]string{"", "", ""},
			true, "no author",
		},
		{
			"no title",
			[]string{val0.Address.String(), "", "bar"},
			true, "no title",
		},
		{
			"no body",
			[]string{val0.Address.String(), "foo", ""},
			true, "no body",
		},
		{
			"valid request",
			[]string{val0.Address.String(), "foo", "bar", fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation)},
			false, "",
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			cmd := cli.CmdCreatePost()
			args := append([]string{
				fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
				fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
				fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(10))).String()),
			}, tc.args...)
			out, err := clitestutil.ExecTestCLICmd(val0.ClientCtx, cmd, args)

			if tc.expErr {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.expErrMsg)
			} else {
				var txRes sdk.TxResponse
				err := val0.ClientCtx.JSONMarshaler.UnmarshalJSON(out.Bytes(), &txRes)
				s.Require().NoError(err)
				s.Require().Equal(uint32(0), txRes.Code)
			}
		})
	}
}

func (s *IntegrationTestSuite) TestAllPosts() {
	val0 := s.network.Validators[0]

	// Create two dummy posts.
	cmd := cli.CmdCreatePost()
	defaultArgs := []string{
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastBlock),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(s.cfg.BondDenom, sdk.NewInt(10))).String()),
	}

	out, err := clitestutil.ExecTestCLICmd(val0.ClientCtx, cmd, append(defaultArgs, val0.Address.String(), "foo1", "bar1"))
	s.Require().NoError(err)
	var txRes1 sdk.TxResponse
	err = val0.ClientCtx.JSONMarshaler.UnmarshalJSON(out.Bytes(), &txRes1)
	s.Require().NoError(err)
	s.Require().Equal(uint32(0), txRes1.Code)
	s.Require().NoError(s.network.WaitForNextBlock())

	out, err = clitestutil.ExecTestCLICmd(val0.ClientCtx, cmd, append(defaultArgs, val0.Address.String(), "foo2", "bar2"))
	s.Require().NoError(err)
	var txRes2 sdk.TxResponse
	err = val0.ClientCtx.JSONMarshaler.UnmarshalJSON(out.Bytes(), &txRes2)
	s.Require().NoError(err)
	s.Require().Equal(uint32(0), txRes2.Code)
	s.Require().NoError(s.network.WaitForNextBlock())

	testCases := []struct {
		name        string
		args        []string
		expErr      bool
		expErrMsg   string
		expNumPosts int
	}{
		{
			"no pagination",
			[]string{"-o=json"},
			false, "", 2,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			cmd := cli.CmdAllPosts()
			out, err := clitestutil.ExecTestCLICmd(val0.ClientCtx, cmd, tc.args)
			if tc.expErr {
				s.Require().Error(err)
				s.Require().Contains(err.Error(), tc.expErrMsg)
			} else {
				var res = blog.QueryAllPostsResponse{}
				err := val0.ClientCtx.JSONMarshaler.UnmarshalJSON(out.Bytes(), &res)
				s.Require().NoError(err)
				s.Require().Equal(tc.expNumPosts, len(res.Posts))
			}
		})
	}
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}
