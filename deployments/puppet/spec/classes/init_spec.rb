require 'spec_helper'

describe 'signalfx_agent' do
  let(:title) { 'signalfx_agent' }
  let(:params) { { 'config' => {} } }
  let (:facts) { { :osfamily => 'redhat'} }

  it "fails without access token" do
    is_expected.to compile.and_raise_error(/signalFxAccessToken/)
  end

  on_supported_os.each do |os, facts|
    context "on #{os}" do
      let(:params) { { 'config' => {
        :signalFxAccessToken => "testing",
      } } }
      let(:facts) do
        facts
      end

      it { is_expected.to compile.with_all_deps }
    end
  end
end
