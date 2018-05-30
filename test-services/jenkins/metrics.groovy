import org.kohsuke.stapler.*
import jenkins.metrics.api.*
import net.sf.json.JSONObject
import org.codehaus.groovy.reflection.*

def metrics = jenkins.model.Jenkins.instance.getExtensionList(jenkins.metrics.api.MetricsAccessKey.DescriptorImpl)[0]

def acckeylist =  jenkins.metrics.api.MetricsAccessKey.DescriptorImpl.class.getDeclaredField("accessKeys");
acckeylist.setAccessible(true);

Object list = new ArrayList<MetricsAccessKey>();
def add = List.class.getDeclaredMethod("add",Object.class);

MetricsAccessKey m = new MetricsAccessKey("aa","33DD8B2F1FD645B814993275703F_EE1FD4D4E204446D5F3200E0F6-C55AC14E",true,false,true,true,"*");

add.invoke(list,m)  
acckeylist.set(metrics,list)